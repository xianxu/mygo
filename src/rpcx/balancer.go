package rpcx

import (
	"gostrich"

	"time"
	"sync"
	"sync/atomic"
	"log"
	"fmt"
	"net/rpc"
)

// TODO: 
//   - Traffic to FLAKY service might be too low
//   - Closer of a service
//   - Service discovery
//
// Generic load balancer logic, provides:
//   - Probing when dead (Supervisor)
//   - Recreate service when dead (Replaceable, Supervisor)
//   - Load balancing among multiple hosts
//

var (
	//TODO: better reporting.
	Timeout              = Error("rpcx.timeout")
	NilUnderlyingService = Error("underlying service is nil")

	// Setting ProberReq to this value, the prober will use last request that triggers a service
	// being marked dead as the prober req.
	ProberReqLastFail ProberReqLastFailType
)

type Error string

func (t Error) Error() string {
	return string(t)
}

type Service interface {
	// a Service, conforming to Go's rpc.Client.
	/*Go(fn string, args interface{}, reply interface{}, done chan *rpc.Call) *Call*/
	/*Call(fn string, args interface{}, reply interface{}) error*/

	// TODO: Should we put a timeout into the interface itself? That would allow at minimal
	//       one less chan creation. Consider the case for rpc, where it already provides
	//       a chan to block on. Without pushing timeout logic down, we'll need to create
	//       another goroutine and channel, just to do timeout.
	//
	// In all seriousness, why would anyone need a service without timeout? Thus include it
	// for now.
	//
	// timeout = 0 means no timeout requested. if a service does not support timeout, it
	// is required caller set timeout to 0, to expressly acknowledge they aren't expecting
	// timeout.
	Serve(req interface{}, rsp interface{}, timeout time.Duration) error

	// TODO: do we want a Closer? seems so.
}

type ServiceMaker interface {
	Make() (name string, s Service, e error)
}

/*
 * Status of a service. We will query bad servers less frequently.
 */
type ServiceStatus int32

const (
	SERVICE_ALIVE = ServiceStatus(0) // Node will be queried normally
	SERVICE_FLAKY = ServiceStatus(1) // Node seems flaky, we will send lower traffic

	// When node seems dead, we dice throw upto 100 times to avoid picking a dead node. Given
	// 10 server cluster and 9 dead, the probability of not picking one dead is > 99.99% after
	// 100 dice. It'll be better in real situation.
	SERVICE_DEAD = ServiceStatus(100)
)

func average(ns []int64) float64 {
	sum := 0.0
	for i := range ns {
		sum += float64(atomic.LoadInt64(&ns[i]))
	}
	return float64(sum) / float64(len(ns))
}

type ServiceReporter interface {
	Report(interface{}, interface{}, error, int64)
}

type ProberReqLastFailType int

// Wrapping on top of a Service, keeping track service history for load balancing purpose.
// There are two strategy dealing with faulty services, either we can keep probing it, or
// we can create a new service to replace the faulty one.
type Supervisor struct {
	svcLock   sync.RWMutex
	service   Service             // underlying service.
	name      string              // Human readable name
	latencies gostrich.IntSampler // Keeps track of host latency, in micro seconds

	status          ServiceStatus // Mutable field of service status
	proberRunning   int32         // Mutable field of whether there's a prober running
	replacerRunning int32         // Mutable field of whether we are replacing service

	// Strategy of dealing with fault, when a service is marked dead.
	//   - To start a prober to probe
	//   - To replace faulty service with a new service.
	//
	// To use prober, set proberReq to anything that's not a ProberReqLastFailType.
	// To replace faulty service, supply a Service Maker.
	//
	// If a serviceMaker is specified, we will replace dead service with one freshly created.
	// This new service keeps old service state. When proberReq is not nil, probing will be
	// started.
	proberReq    interface{}
	serviceMaker ServiceMaker

	reporter ServiceReporter // where to report service status, gostrich thing

	// the following are specific limit of what's considered flaky/dead for the service
	flaky float64 // What's average latency to be considered flaky in micro
	dead  float64 // What's average latency to be considered dead in micro

	// frequency to run prober and replacer when service gone dead.
	proberFreqSec   byte
	replacerFreqSec byte
}

// A Balancer is supervisor that tracks last 100 service call status. It recovers mostly by keep
// probing. In rare situation, ServiceMaker may be invoked to recreate all underlying services,
// but that sounds "dangerous" and "expensive".
func NewSupervisor(name string, service Service, reporter ServiceReporter, proberReq interface{}, serviceMaker ServiceMaker) *Supervisor {
	return &Supervisor{
		sync.RWMutex{},
		service,
		name,
		gostrich.NewIntSampler(100),
		SERVICE_ALIVE,
		0,
		0,
		proberReq,
		serviceMaker,
		reporter,
		1000 * 1000,
		9500 * 1000,
		3,
		60,
	}
}

// A replaceable service that recovers from error by replacing underlying service with a new one
// from service maker.
func NewReplaceable(name string, service Service, reporter ServiceReporter, serviceMaker ServiceMaker) *Supervisor {
	return &Supervisor{
		sync.RWMutex{},
		service,
		name,
		gostrich.NewIntSampler(2),
		SERVICE_ALIVE,
		0,
		0,
		nil,
		serviceMaker,
		reporter,
		4000 * 1000, // one failure reduces chance of this channel being used.
		9000 * 1000, // two failure result it being replaced.
		3,           // not used
		3,           // at replacer level, we try to re-establish service every 3 seconds
	}
}

func MicroTilNow(then time.Time) int64 {
	now := time.Now()
	return int64((now.Second()-then.Second())*1000000 + (now.Nanosecond()-then.Nanosecond())/1000)
}

func (s *Supervisor) Serve(req interface{}, rsp interface{}, timeout time.Duration) (err error) {
	then := time.Now()
	s.svcLock.RLock()
	if s.service == nil {
		err = NilUnderlyingService
	} else {
		err = s.service.Serve(req, rsp, timeout)
	}

	// micro seconds
	latency := MicroTilNow(then)

	// collect stats before adjusting latency
	if s.reporter != nil {
		s.reporter.Report(req, rsp, err, latency)
	}

	// adjust latency on failure to be 10 sec
	if err != nil {
		latency = 10 * 1000000
	}

	var avg float64
	if s.service != nil {
		// only need to keep track of history if service's not nil
		s.latencies.Observe(latency)
		avg = average(s.latencies.Sampled())
	} else {
		// otherwise treat as dead
		avg = s.dead
	}
	s.svcLock.RUnlock()

	// change service state
	switch {
	case avg >= s.dead:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_DEAD))
		// Reactions to service being dead:
		// If we have serviceMaker, try make a new service out of it.
		if s.serviceMaker != nil {
			if atomic.CompareAndSwapInt32((*int32)(&s.replacerRunning), 0, 1) {
				// TODO: do we want to set underlying service to nil? If we decide it's a good
				// time to replace underlying service, we pretty much have decided the error's
				// not recoverable? Setting it to nil would allow failing fast for all subsequent
				// calls to this service, until we recover.
				s.svcLock.Lock()
				s.service = nil
				s.svcLock.Unlock()
				go func() {
					log.Printf("Service \"%v\" gone bad, start replacer routine\n", s.name)
					for {
						_, newService, err := s.serviceMaker.(ServiceMaker).Make()
						if err == nil {
							log.Printf("replacer obtained new service for \"%v\"", s.name, err)
							s.svcLock.Lock()
							s.service = newService
							s.latencies.Clear()
							atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_ALIVE))
							atomic.StoreInt32((*int32)(&s.replacerRunning), 0)
							s.svcLock.Unlock()
							log.Printf("replacer exited \"%v\"", s.name, err)
							break
						} else {
							log.Printf("replacer errors out for \"%v\", will try later. %v", s.name, err)
						}

						time.Sleep(time.Duration(int64(s.replacerFreqSec)) * time.Second)
					}
				}()
			}
		}
		// if we have a prober, try probe it.
		if s.proberReq != nil {
			if atomic.CompareAndSwapInt32((*int32)(&s.proberRunning), 0, 1) {
				go func() {
					log.Printf("Service \"%v\" gone bad, start probing\n", s.name)
					for {
						time.Sleep(time.Duration(int64(s.proberFreqSec)) * time.Second)
						log.Printf("Service %v is dead, probing..", s.name)
						switch s.proberReq.(type) {
						case ProberReqLastFailType:
							s.Serve(req, rsp, timeout)
						default:
							s.Serve(s.proberReq, rsp, timeout)
						}
						if atomic.LoadInt32((*int32)(&s.status)) < int32(SERVICE_DEAD) {
							log.Printf("Service \"%v\" recovered, exit prober routine\n", s.name)
							atomic.StoreInt32((*int32)(&s.proberRunning), 0)
							break
						}
					}
				}()
			}
		}
	case avg > s.flaky:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_FLAKY))
	default:
		atomic.StoreInt32((*int32)(&s.status), int32(SERVICE_ALIVE))
	}
	return
}

// Wrapper of a service that times out. Typically it's more efficiently to rely on underlying
// service's timeout mechanism.
type ServiceWithTimeout struct {
	Service Service
	Timeout time.Duration
}

func (s *ServiceWithTimeout) Serve(req interface{}, rsp interface{}, timeout time.Duration) (err error) {
	tick := time.After(s.Timeout)
	// need at least 1 capacity so that when a rpc call return after timeout has occurred,
	// it doesn't block the goroutine sending such notification.
	// not sure why rpc package uses capacity 10 though.
	done := make(chan error, 1)

	go func() {
		err = s.Service.Serve(req, rsp, timeout)
		done <- err
	}()

	select {
	case <-done:
	case <-tick:
		err = Timeout
	}
	return
}

/*
 * A cluster represents a load balanced set of services. At higher level, it can be load balanced
 * services across multiple machines. At lower level, it can also be used to manage connection
 * pool to a single host.
 *
 * TODO: the logic to grow and shrink the service pool is not implemented yet.
 */
type Cluster struct {
	Name     string
	Services []*Supervisor
	Retries  int             // if there's failure, retry another host
	Reporter ServiceReporter // stats reporter of how cluster, rolled up from each host

	// internals, default values' fine
	Lock sync.RWMutex // guard services
}

func (c *Cluster) serveOnce(req interface{}, rsp interface{}, timeout time.Duration) (err error) {
	c.Lock.RLock()
	defer c.Lock.RUnlock()

	then := time.Now()

	if len(c.Services) == 0 {
		err = Error("There's no underlying service in cluster " + c.Name)
		return
	}

	// pick one random
	// ... Nanosecond() on linux only have precision of microsecond, thus use microsecond as dice
	s := c.Services[(time.Now().Nanosecond()/1000)%len(c.Services)]

	// adjust the pick by the healthiness of the node
	for retries := 0; atomic.LoadInt32((*int32)(&s.status)) > int32(retries); retries += 1 {
		s = c.Services[(time.Now().Nanosecond()/1000)%len(c.Services)]
	}
	err = s.Serve(req, rsp, timeout)

	if c.Reporter != nil {
		latency := MicroTilNow(then)
		c.Reporter.Report(req, rsp, err, latency)
	}

	return
}

func (c *Cluster) Serve(req interface{}, rsp interface{}, timeout time.Duration) (err error) {
	for i := 0; i <= c.Retries; i += 1 {
		err = c.serveOnce(req, rsp, timeout)
		if err == nil {
			return
		} else {
			log.Printf("Error serving request in cluster %v. Error is: %v\n", c.Name, err)
		}
	}
	log.Printf("Exhausted retries of serving request in cluster %v\n", c.Name)
	return
}

type BasicStatsReporter struct {
	counterReq, counterSucc, counterFail, counterRspNil gostrich.Counter
	reqLatencyStat                                      gostrich.IntSampler
}

func NewBasicStatsReporter(stats gostrich.Stats) *BasicStatsReporter {
	return &BasicStatsReporter{
		counterReq:     stats.Counter("req"),
		counterSucc:    stats.Counter("req/success"),
		counterFail:    stats.Counter("req/fail"),
		reqLatencyStat: stats.Statistics("req/latency"),
		counterRspNil:  stats.Counter("rsp/nil"),
	}
}

func (r *BasicStatsReporter) Report(req interface{}, rsp interface{}, err error, micro int64) {
	r.reqLatencyStat.Observe(micro)
	r.counterReq.Incr(1)
	if err != nil {
		r.counterFail.Incr(1)
	} else {
		r.counterSucc.Incr(1)
		if rsp == nil {
			r.counterRspNil.Incr(1)
		}
	}
}

// Wrap rpc call service name and arguments in a single struct, to be used with a Service as
// request.
type RpcReq struct {
	Fn   string
	Args interface{}
}
type RpcCaller interface {
	Call(method string, request interface{}, response interface{}) error
}
type RpcGoer interface {
	Go(method string, request interface{}, response interface{}, done chan *rpc.Call) *rpc.Call
}
type RPCClient interface {
	RpcCaller
	RpcGoer
}

// Using cassandra as an example, typical setup is:
//
//                                  /  Replaceable - KeyspaceService  --\
//            Supervisor - Cluster  -  Replaceable - KeyspaceService  ---->  Cassandra host 1
//          /                       \  Replaceable - KeyspaceService  --/
//         / 
// Cluster  
//         \
//          \                       /  Replaceable - KeyspaceService  --\
//            Supervisor - Cluster  -  Replaceable - KeyspaceService  ---->  Cassandra host 2
//                                  \  Replaceable - KeyspaceService  --/
//

// Create a reliable service out of a group of service makers. n services will be make from
// each ServiceMaker (think connections).
type ReliableService struct {
	Name        string
	Makers      []ServiceMaker // ClientBuilder
	Retries     int            // default to 0
	Concurrency int            // default to 1
	Prober      interface{}    // default to nil
	Stats       gostrich.Stats // default to nil
}

func NewReliableService(conf ReliableService) Service {
	supers := make([]*Supervisor, len(conf.Makers))

	var sname string
	var svc Service
	var err error
	var reporter ServiceReporter

	for i, maker := range conf.Makers {

		var concur int
		if conf.Concurrency == 0 {
			concur = 1
		} else {
			concur = conf.Concurrency
		}
		services := make([]*Supervisor, concur)
		for j := range services {
			sname, svc, err = maker.Make()
			if err != nil {
				log.Printf("Failed to make a service: %v %v. Error is %v", conf.Name, sname, err)
			}
			services[j] = NewReplaceable(fmt.Sprintf("%v:conn:%v", conf.Name, j), svc, nil, maker)
		}
		cluster := &Cluster{Name: sname, Services: services}
		if conf.Stats != nil {
			reporter = NewBasicStatsReporter(conf.Stats.Scoped(sname))
		}
		supers[i] = NewSupervisor(sname, cluster, reporter, conf.Prober, nil)
	}

	if conf.Stats != nil {
		reporter = NewBasicStatsReporter(conf.Stats)
	}
	return &Cluster{
		Name:     conf.Name,
		Services: supers,
		Retries:  conf.Retries,
		Reporter: reporter,
	}
}
