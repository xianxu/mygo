package tfe

/*
 * A simple front end server that proxies request, as in Tfe
 */
import (
	"gostrich"

	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"code.google.com/p/log4go"
)

var (
	TfeTimeout = TfeError("timeout")
)

// bunch of alias to make rule writing more descriptive
type RuleName string
type RequestHost string
type RequestPrefix string
type ProxiedPrefix string
type ProxiedHost string
type Retries int
type Timeout time.Duration
type MaxIdleConnsPerHost int

type TfeError string

func (t TfeError) Error() string {
	return string(t)
}

type Rules []Rule

/*
* A Tfe is basically set of rules handled by it. The set of rules is expressed as port number to
* a list of rules.
 */
type Tfe struct {
	// Note: rules can't be added dynamically for now.
	BindingToRules map[string]Rules
}

/*
* A routing rule encapsulates routes to a service, it provides three things:
*   - to determine whether a rule can be applied to a given request
*   - to transform request before forwarding to downstream
*   - to transform response before replying upstream
*
* And then it provides some configuration values on a per service level, such as how
* many times to retry for transport failures and what's the timeout for that service.
*
* It also provides a way to report stats on the service.
 */
type Rule interface {
	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request and return a specific downstream client to connect to
	TransformRequest(*http.Request) *TransportWithHost

	// mutate response
	TransformResponse(*http.Response)

	// retries
	GetClientRetries() int

	// timeouts
	GetClientTimeout() time.Duration

	// function to gather stats
	GatherStats() func(*http.Request, *http.Response, error, int)
}

/*
 * Status of a node. We will query bad servers less frequently.
 */
type NodeStatus int32

const (
	NODE_ALIVE = NodeStatus(0) // node will be queried normally
	NODE_FLAKY = NodeStatus(1) // node seems flaky, we will send lower traffic
	NODE_DEAD  = NodeStatus(8) // node seems dead, we will send even lower traffic
)

/*
* Keep tracks of host, connection pool for each host and latency history. Such information's
* used for load balancing.
 */
type TransportWithHost struct {
	hostPort  string
	transport http.Transport
	latencies gostrich.IntSampler // keeps track of host latency, in micro seconds

	status NodeStatus // mutable field

	flaky float64 // what's average latency to be considered flaky in micro
	dead  float64 // what's average latency to be considered dead in micro

	proberRunning int32 // whether there's a prober started already
	gatherStats   func(*http.Request, *http.Response, error, int)
}

// Since we treat failures as latency 10 second, if keeping track of 100 points, 
// 10% bad to react, means to react if average > 1 sec. To be viewed as dead,
// require 95% failures. In flaky mode, with or without prober, when downstream becomes
// healthy, we'd discover it. In dead mode though, prober's needed to bring it back. At 95% mark
// dead rate, it takes about 5 success probes to bring server back to flaky state. From there, 
// recovery should be fast.
func NewTransportWithHost(host string, hl MaxIdleConnsPerHost) *TransportWithHost {
	return &TransportWithHost{
		host,
		http.Transport{MaxIdleConnsPerHost: int(hl), DisableCompression: true}, // transport to use
		gostrich.NewIntSampler(100),
		NODE_ALIVE,
		1000 * 1000,
		9500 * 1000,
		0,
		nil,
	}
}
/*
* Simple rule implementation that allows filter based on Host/port and resource prefix.
 */
type prefixRewriteRule struct {
	name string
	// transformation rules
	sourceHost           string // "" matches all
	sourcePathPrefix     string
	proxiedPathPrefix    string
	proxiedAttachHeaders map[string][]string

	// clients to use
	clients       []*TransportWithHost
	clientRetries int
	clientTimeout time.Duration

	// internals
	clientsLock sync.RWMutex // guards mutation to Clients field.

	// counters
	gatherStats func(*http.Request, *http.Response, error, int)
}

func gatherRequestStats(stats gostrich.Stats) func(*http.Request, *http.Response, error, int) {
	counterReq := stats.Counter("req")
	counterSucc := stats.Counter("req/success")
	counterFail := stats.Counter("req/fail")
	LatencyStat := stats.Statistics("req/latency")

	counter1xx := stats.Counter("rsp/1xx")
	counter2xx := stats.Counter("rsp/2xx")
	counter3xx := stats.Counter("rsp/3xx")
	counter4xx := stats.Counter("rsp/4xx")
	counter5xx := stats.Counter("rsp/5xx")
	counterRst := stats.Counter("rsp/rst")

	return func(req *http.Request, rsp *http.Response, err error, micro int) {
		counterReq.Incr(1)
		if err != nil {
			counterFail.Incr(1)
		} else {
			counterSucc.Incr(1)
			code := rsp.StatusCode
			switch {
			case code >= 100 && code < 200:
				counter1xx.Incr(1)
			case code >= 200 && code < 300:
				counter2xx.Incr(1)
			case code >= 300 && code < 400:
				counter3xx.Incr(1)
			case code >= 400 && code < 500:
				counter4xx.Incr(1)
			case code >= 500 && code < 600:
				counter5xx.Incr(1)
			default:
				counterRst.Incr(1)
			}
		}
		LatencyStat.Observe(micro)
	}
}

func NewPrefixRule(rn RuleName,
rh RequestHost,
rp RequestPrefix,
pp ProxiedPrefix,
pah map[string][]string,
c []*TransportWithHost,
r Retries,
to Timeout) *prefixRewriteRule {
	for _, t := range c {
		if t.gatherStats == nil {
			t.gatherStats = gatherRequestStats(
				gostrich.StatsSingleton().Scoped(string(rn)).Scoped(t.hostPort))
		}
	}
	return &prefixRewriteRule{
		string(rn),
		string(rh),
		string(rp),
		string(pp),
		pah,
		c,
		int(r),
		time.Duration(to),
		sync.RWMutex{},
		gatherRequestStats(gostrich.StatsSingleton().Scoped(string(rn))),
	}
}

func (p *prefixRewriteRule) HandlesRequest(r *http.Request) bool {
	return (p.sourceHost == "" || p.sourceHost == r.Host) &&
		strings.HasPrefix(r.URL.Path, p.sourcePathPrefix)
}

func (p *prefixRewriteRule) TransformRequest(r *http.Request) *TransportWithHost {
	p.clientsLock.RLock()
	defer p.clientsLock.RUnlock()

	if len(p.clients) == 0 {
		return nil
	}

	// pick one random
	client := p.clients[time.Now().Nanosecond()%len(p.clients)]

	// adjust the pick by the healthiness of the node
	for retries := 0; atomic.LoadInt32((*int32)(&client.status)) > int32(retries); retries += 1 {
		client = p.clients[time.Now().Nanosecond()%len(p.clients)]
	}
	r.URL = &url.URL{
		"http", //TODO: shit why this is not set?
		r.URL.Opaque,
		r.URL.User,
		client.hostPort,
		p.proxiedPathPrefix + r.URL.Path[len(p.sourcePathPrefix):len(r.URL.Path)],
		r.URL.RawQuery,
		r.URL.Fragment,
	}
	r.Host = client.hostPort
	r.RequestURI = ""
	for k, v := range p.proxiedAttachHeaders {
		r.Header[k] = v
	}
	return client
}

func (p *prefixRewriteRule) TransformResponse(rsp *http.Response) {
	//TODO
	return
}

func (p *prefixRewriteRule) GetClientRetries() int {
	return p.clientRetries
}

func (p *prefixRewriteRule) GetClientTimeout() time.Duration {
	return p.clientTimeout
}

func (p *prefixRewriteRule) GatherStats() func(*http.Request, *http.Response, error, int) {
	return p.gatherStats
}

func (p *TransportWithHost) GatherStats() func(*http.Request, *http.Response, error, int) {
	return p.gatherStats
}

type intSlice []int

func (ns intSlice) average() float64 {
	sum := 0.0
	for _, v := range ns {
		sum += float64(v)
	}
	return float64(sum) / float64(len(ns))
}

type responseAndError struct {
	rsp *http.Response
	err error
}

func requestWithProber(rule Rule, client *TransportWithHost, req *http.Request,
dt time.Duration) (rsp *http.Response, err error) {
	rsp = nil

	tick := time.After(dt)
	done := make(chan *responseAndError)
	then := time.Now()

	go func() {
		rsp, err = client.transport.RoundTrip(req)
		done <- &responseAndError{rsp, err}
	}()

	select {
	case rae := <-done:
		rsp = rae.rsp
		err = rae.err
	case <-tick:
		err = TfeTimeout
	}

	now := time.Now()

	// micro seconds
	latency := (now.Second()-then.Second())*1000000 +
		(now.Nanosecond()-then.Nanosecond())/1000

		// collect stats before adjusting latency
	rule.GatherStats()(req, rsp, err, latency)
	client.GatherStats()(req, rsp, err, latency)

	if err != nil {
		latency = 10 * 1000000
	}

	client.latencies.Observe(latency)
	avg := intSlice(client.latencies.Sampled()).average()

	// change client state
	switch {
	case avg > client.dead:
		atomic.StoreInt32((*int32)(&client.status), int32(NODE_DEAD))
		// start prober to probe dead node, if there's no prober running
		if atomic.CompareAndSwapInt32((*int32)(&client.proberRunning), 0, 1) {
			go func() {
				// probe every 1 second
				for {
					time.Sleep(1 * time.Second)
					requestWithProber(rule, client, req, dt)
					if atomic.LoadInt32((*int32)(&client.status)) < int32(NODE_DEAD) {
						break
					}
				}
			}()
		}
	case avg > client.flaky:
		atomic.StoreInt32((*int32)(&client.status), int32(NODE_FLAKY))
	default:
		atomic.StoreInt32((*int32)(&client.status), int32(NODE_ALIVE))
	}

	return
}

func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {

			var rsp *http.Response
			var err error
			// retry till we get nil err, or else
			for tried := 0; (tried == 0 || err != nil) && tried <= rule.GetClientRetries(); tried += 1 {
				client := rule.TransformRequest(r)

				if client == nil {
					log4go.Warn("No client is available to handle request %v\n", *r)
					w.WriteHeader(503)
					return
				}

				rsp, err = requestWithProber(rule, client, r, rule.GetClientTimeout())
			}

			if err != nil {
				log4go.Warn("Error occurred while proxying: %v\n", err.Error())
				w.WriteHeader(503)
				return
			}

			rule.TransformResponse(rsp)

			headers := w.Header()
			for k, v := range rsp.Header {
				headers[k] = v
			}
			w.WriteHeader(rsp.StatusCode)

			if rsp != nil && rsp.StatusCode >= 300 && rsp.StatusCode < 400 {
				// if redirects
				return
			}

			// 2XX with body
			_, err = io.Copy(w, rsp.Body)

			// log error while copying
			if err != nil {
				log4go.Warn("err while piping bytes: %v\n", err)
			}

			return
		}
	}

	w.WriteHeader(404)
	return
}
