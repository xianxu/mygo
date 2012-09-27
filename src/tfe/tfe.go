package tfe

/*
 * A simple front end server that proxies request, as in Tfe
 */
import (
	"gostrich"

	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"sync"
	"time"
)

var (
	TfeTimeout = TfeError("timeout")
)

type RuleName            string
type RequestHost         string
type RequestPrefix       string
type ProxiedPrefix       string
type ProxiedHost         string
type Retries             int
type Timeout             time.Duration
type MaxIdleConnsPerHost int

type TfeError string
func (t TfeError) Error() string{
	return string(t)
}

type Rules []Rule

/*
* A Tfe is basically set of rules handled by it. The set of rules is expressed as port number to
* a list of rules.
*/
type Tfe struct {
	BindingToRules map[string]Rules
}

/*
* A routing rule means three things:
*   - to determine whether a rule can be applied to a given request
*   - to transform request before forwarding to downstream
*   - to transform response before replying upstream
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
	NODE_ALIVE = NodeStatus(0)  // node will be queried normally
	NODE_FLAKY = NodeStatus(1)	// node seems flaky, we will send lower traffic
	NODE_DEAD  = NodeStatus(8)	// node seems dead, we will send even lower traffic
)

/*
* Keep tracks of host, connection pool for each host and latency history. Will do meaningful
* balancing later but for now we'd just round robin.
*/
type TransportWithHost struct {
	hostPort      string
	transport     http.Transport
	latencies     gostrich.Sampler  // keeps track of host latency

	status        NodeStatus        // mutable field
	flaky	      float64           // what's average latency to be considered flaky in millisecond
	dead          float64           // what's average latency to be considered dead in millisecond
	proberRunning int32             // whether there's a prober started already
    gatherStats   func(*http.Request, *http.Response, error, int)
}

//TODO: we will need to tweak numbers, only keeping track of 10 seems too reactive?
//
// Make a new host with construct to keep track of its health. Internally we keep last 10 call's
// latency and treat failures as taking 10 seconds. This means a single error will make average
// latency greater than 1 second. If we want to react to 5 errors out of last 10 requests, we'd
// set flaky to 5000 and dead to 10000, probably.
//
// if keeping track of 100 points, 10% bad to react, means to react if average > 1 sec. to be viewed
// as dead, require 95% failures. In flaky mode, with or without prober, when downstream becomes
// healthy, we'd discover it. in dead mode though, prober's needed to bring it back. at 95% mark
// dead rate, it takes about 5 success probes to bring server back to flaky state. From there, 
// recovery should be fast.
func NewTransportWithHost(host string, hl MaxIdleConnsPerHost) *TransportWithHost {
	return &TransportWithHost {
		host,
		http.Transport{ MaxIdleConnsPerHost: int(hl) },            // transport to use
		gostrich.NewSampler(100),
		NODE_ALIVE,
		1000 * 1000,
		9500 * 1000,
		0,
		nil, //gatherRequestStats(gostrich.StatsSingleton.Scoped(tfeLastRuleName).Scoped(host)),
	}
}
/*
* Simple rule implementation that allows filter based on Host/port and resource prefix.
*/
type PrefixRewriteRule struct {
	Name                 string
	// transformation rules
	SourceHost           string        // "" matches all
	SourcePathPrefix     string
	ProxiedPathPrefix    string
	ProxiedAttachHeaders map[string][]string

	// clients to use
	Clients              []*TransportWithHost
	ClientRetries        int
	ClientTimeout        time.Duration

	// internals
	clientsLock          sync.RWMutex  // guards mutation to Clients field.

	// counters
    gatherStats          func(*http.Request, *http.Response, error, int)
}

func gatherRequestStats(stats gostrich.Stats) func(*http.Request, *http.Response, error, int) {
	fmt.Println("DEB: setting up stats gathering")
	counterReq  := stats.Counter("req")
	counterSucc := stats.Counter("req/success")
	counterFail := stats.Counter("req/fail")
	LatencyStat := stats.Statistics("req/latency")

	counter1xx  := stats.Counter("rsp/1xx")
	counter2xx  := stats.Counter("rsp/2xx")
	counter3xx  := stats.Counter("rsp/3xx")
	counter4xx  := stats.Counter("rsp/4xx")
	counter5xx  := stats.Counter("rsp/5xx")
	counterRst  := stats.Counter("rsp/rst")

	return func(req *http.Request, rsp *http.Response, err error, micro int) {
		fmt.Println("Gathering stats")
		counterReq.Incr(1)
		if err != nil {
			counterFail.Incr(1)
		} else {
			counterSucc.Incr(1)
			code := rsp.StatusCode
			switch {
			case  code >= 100 && code < 200:
				counter1xx.Incr(1)
			case  code >= 200 && code < 300:
				counter2xx.Incr(1)
			case  code >= 300 && code < 400:
				counter3xx.Incr(1)
			case  code >= 400 && code < 500:
				counter4xx.Incr(1)
			case  code >= 500 && code < 600:
				counter5xx.Incr(1)
			default:
				counterRst.Incr(1)
			}
		}
		LatencyStat.Observe(float64(micro))
	}
}

func NewPrefixRule(
	rn RuleName,
	rh RequestHost,
	rp RequestPrefix,
	pp ProxiedPrefix,
	pah map[string][]string,
	c []*TransportWithHost,
	r Retries,
	to Timeout) *PrefixRewriteRule {
	for _, t := range c {
		if t.gatherStats == nil {
			t.gatherStats = gatherRequestStats(
				gostrich.StatsSingleton.Scoped(string(rn)).Scoped(t.hostPort))
		}
	}
	return &PrefixRewriteRule {
		string(rn),
		string(rh),
		string(rp),
		string(pp),
		pah,
		c,
		int(r),
		time.Duration(to),
		sync.RWMutex{},
		gatherRequestStats(gostrich.StatsSingleton.Scoped(string(rn))),
	}
}

func (p *PrefixRewriteRule) HandlesRequest(r *http.Request) bool {
	return (p.SourceHost == "" || p.SourceHost == r.Host) &&
		strings.HasPrefix(r.URL.Path, p.SourcePathPrefix)
}

func (p *PrefixRewriteRule) TransformRequest(r *http.Request) *TransportWithHost {
	p.clientsLock.RLock()
	defer p.clientsLock.RUnlock()

	if len(p.Clients) == 0 {
		return nil
	}

	// pick one random
	client := p.Clients[time.Now().Nanosecond()%len(p.Clients)]

	// adjust the pick by the healthiness of the node
	for retries := 0; atomic.LoadInt32((*int32)(&client.status)) > int32(retries); retries += 1 {
		client = p.Clients[time.Now().Nanosecond()%len(p.Clients)]
	}
	r.URL = &url.URL {
		"http",  //TODO: shit why this is not set?
		r.URL.Opaque,
		r.URL.User,
		client.hostPort,
		p.ProxiedPathPrefix + r.URL.Path[len(p.SourcePathPrefix):len(r.URL.Path)],
		r.URL.RawQuery,
		r.URL.Fragment,
	}
	r.Host = client.hostPort
	r.RequestURI = ""
	for k, v := range p.ProxiedAttachHeaders {
		r.Header[k] = v
	}
	return client
}

func (p *PrefixRewriteRule) TransformResponse(rsp *http.Response) {
	//TODO
	return
}

func (p *PrefixRewriteRule) GetClientRetries() int {
	return p.ClientRetries
}

func (p *PrefixRewriteRule) GetClientTimeout() time.Duration {
	return p.ClientTimeout
}

func (p *PrefixRewriteRule) GatherStats() func(*http.Request, *http.Response, error, int) {
	return p.gatherStats
}

func (p *TransportWithHost) GatherStats() func(*http.Request, *http.Response, error, int) {
	return p.gatherStats
}

type float64Slice []float64
func (ns float64Slice) average() float64 {
	sum := 0.0
	for _, v := range ns {
		sum += v
	}
	return sum / float64(len(ns))
}

type responseAndError struct {
	rsp *http.Response
	err error
}

func requestWithProber(
	rule Rule, client *TransportWithHost, req *http.Request,
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

	now :=  time.Now()

	// micro seconds
	latency := (now.Second() - then.Second()) * 1000000 +
		(now.Nanosecond() - then.Nanosecond()) / 1000

	// collect stats before adjusting latency
	rule.GatherStats()(req, rsp, err, latency)
	client.GatherStats()(req, rsp, err, latency)

	if err != nil {
		latency = 10 * 1000000
	}

	client.latencies.Observe(float64(latency))
	avg := float64Slice(client.latencies.Sampled()).average()

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
	fmt.Printf("Client status is: %v\n", client.status)
	fmt.Printf("Latencies avg: %v\n", avg)
	fmt.Printf("%v", client.latencies.Sampled())

	return
}

func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {
			fmt.Print("Routes detected\n")

			var rsp *http.Response
			var err error
			// retry till we get nil err, or else
			for tried := 0;
			    (tried == 0 || err != nil) && tried <= rule.GetClientRetries();
				tried += 1 {
				client := rule.TransformRequest(r)

				if client == nil {
					fmt.Println("No client is available")
					w.WriteHeader(503)
					return
				}

				fmt.Print(fmt.Sprintf("And host's transformed to %v\n", r.URL.Host))
				fmt.Print(fmt.Sprintf("And path's transformed to %v\n", r.URL.Path))
				fmt.Print(fmt.Sprintf("And header's transformed to %v\n", r.Header))
				fmt.Print(fmt.Sprintf("%v\n", r))

				rsp, err = requestWithProber(rule, client, r, rule.GetClientTimeout())
			}

			if err != nil {
				fmt.Println("Error occurred while proxying: " + err.Error())
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
			bytes, err := io.Copy(w, rsp.Body)

			// log error while copying
			fmt.Printf("%v bytes moved\n", bytes)
			if err != nil {
				fmt.Printf("err while reading: %v\n", err)
			}

			return
		}
	}

	w.WriteHeader(404)
	return
}
