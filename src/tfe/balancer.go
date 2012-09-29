package tfe

import (
	"gostrich"

	"time"
	"net/http"
	"sync"
	"sync/atomic"
	"log"
)
/*
 * Generic load balancer/
 */

var (
	TfeTimeout = TfeError("timeout")
)

type TfeError string

func (t TfeError) Error() string {
	return string(t)
}

type Service interface {
	Serve(req interface{})(rsp interface{}, err error)
}

type HttpService struct {
	http.RoundTripper
	HostPort string  // this service will rewrite request to this host port and binds allawable
	                 // request
}

func (h *HttpService) Serve(req interface{})(rsp interface{}, err error) {
	httpReq := req.(*http.Request)

	httpReq.URL.Scheme = "http"  // hack?
	httpReq.URL.Host = h.HostPort
	httpReq.Host = h.HostPort
	httpReq.RequestURI = ""

	return h.RoundTrip(httpReq)
}

/*
 * Status of a service. We will query bad servers less frequently.
 */
type ServiceStatus int32

const (
	SERVICE_ALIVE = ServiceStatus(0) // node will be queried normally
	SERVICE_FLAKY = ServiceStatus(1) // node seems flaky, we will send lower traffic
	SERVICE_DEAD  = ServiceStatus(8) // node seems dead, we will send even lower traffic
)

type intSlice []int

func (ns intSlice) average() float64 {
	sum := 0.0
	for _, v := range ns {
		sum += float64(v)
	}
	return float64(sum) / float64(len(ns))
}

type ServiceReporter func(interface{}, interface{}, error, int)

type ServiceWithHistory struct {
	service       Service
	name          string              // human readable name
	latencies     gostrich.IntSampler // keeps track of host latency, in micro seconds

	status        ServiceStatus       // mutable field of service status
	proberRunning int32	              // mutable field of whether there's a prober running

	reportTo      ServiceReporter
	flaky         float64 // what's average latency to be considered flaky in micro
	dead          float64 // what's average latency to be considered dead in micro
}

func NewServiceWithHistory(service Service, name string, reportTo ServiceReporter) *ServiceWithHistory {
	return &ServiceWithHistory {
		service,
		name,
		gostrich.NewIntSampler(100),
		SERVICE_ALIVE,
		0,
		reportTo,
		1000 * 1000,
		9500 * 1000,
	}
}

func (s *ServiceWithHistory) Serve(req interface{})(rsp interface{}, err error) {
	then := time.Now()
	rsp, err = s.service.Serve(req)
    now := time.Now()
	// micro seconds
	latency := (now.Second()-then.Second())*1000000 +
		(now.Nanosecond()-then.Nanosecond())/1000

	// collect stats before adjusting latency
	if s.reportTo != nil {
		s.reportTo(req, rsp, err, latency)
	}

	if err != nil {
		latency = 10 * 1000000
	}

	s.latencies.Observe(latency)
	avg := intSlice(s.latencies.Sampled()).average()

	// change service state
	switch {
	case avg > s.dead:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_DEAD))
		// start prober to probe dead node, if there's no prober running
		if atomic.CompareAndSwapInt32((*int32)(&s.proberRunning), 0, 1) {
			go func() {
				log.Printf("Service %v gone bad, start probing\n", s.name)
				// probe every 1 second
				for {
					time.Sleep(1 * time.Second)
					log.Printf("Service %v is dead, probing..", s.name)
					s.Serve(req)  // don't care about result
					if atomic.LoadInt32((*int32)(&s.status)) < int32(SERVICE_DEAD) {
						log.Printf("Service %v recovered\n", s.name)
						break
					}
				}
			}()
		}
	case avg > s.flaky:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_FLAKY))
	default:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_ALIVE))
	}
	return
}

type serviceWithTimeout struct {
	service       Service
	timeout       time.Duration
}

func NewServiceWithTimeout(s Service, timeout time.Duration) *serviceWithTimeout {
	return &serviceWithTimeout {
		s,
		timeout,
	}
}

type responseAndError struct {
	rsp interface{}
	err error
}

func (s *serviceWithTimeout) Serve(req interface{})(rsp interface{}, err error) {
	rsp = nil

	tick := time.After(s.timeout)
	done := make(chan *responseAndError)

	go func() {
		rsp, err = s.service.Serve(req)
		done <- &responseAndError{rsp, err}
	}()

	select {
	case rae := <-done:
		rsp = rae.rsp
		err = rae.err
	case <-tick:
		err = TfeTimeout
	}
	return
}

/*
 * cluster can be used where host/port information itself is not in the request
 */
type cluster struct {
	services []*ServiceWithHistory
	name     string
	tries  int
	reportTo ServiceReporter
	lock     sync.RWMutex
}

type LoadBalancer interface {
	Serve(req interface{})(rsp interface{}, err error)
}

func (c *cluster) serveOnce(req interface{})(rsp interface{}, err error) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	rsp = nil
	err = nil
	then := time.Now()

	if len(c.services) == 0 {
		err = TfeError("There's no underlying service in cluster " + c.name)
		return
	}

	// pick one random
	s := c.services[time.Now().Nanosecond()%len(c.services)]

	// adjust the pick by the healthiness of the node
	for retries := 0; atomic.LoadInt32((*int32)(&s.status)) > int32(retries); retries += 1 {
		s = c.services[time.Now().Nanosecond()%len(c.services)]
	}
	rsp, err = s.Serve(req)

    now := time.Now()
	// micro seconds
	latency := (now.Second()-then.Second())*1000000 +
		(now.Nanosecond()-then.Nanosecond())/1000

	// collect stats before adjusting latency
	if c.reportTo != nil {
		c.reportTo(req, rsp, err, latency)
	}

	return
}

func (c *cluster) Serve(req interface{})(rsp interface{}, err error) {
	for i := 0; i < c.tries; i += 1 {
		rsp, err = c.serveOnce(req)
		if err != nil {
			return
		}
	}
	return
}

func NewCluster(services []*ServiceWithHistory, name string, tries int, reportTo ServiceReporter) *cluster {
	return &cluster {
		services,
		name,
		tries,
		reportTo,
		sync.RWMutex {},
	}
}

