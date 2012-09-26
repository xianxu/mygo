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
	"time"
)

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
}

/*
 * Status of a node. We will query bad servers less frequently.
 */
type NodeStatus int32
const (
	NODE_ALIVE = NodeStatus(0)  // node will be queried normally
	NODE_FLAKY = NodeStatus(1)	// node seems flaky, we will send lower traffic
	NODE_DEAD  = NodeStatus(8)	// node seems dead, we will send even lower traffic
	//TODO permanently dead? avoid herding
)

/*
* Keep tracks of host, connection pool for each host and latency history. Will do meaningful
* balancing later but for now we'd just round robin.
*/
type TransportWithHost struct {
	hostPort      string
	transport     http.Transport
	latencies     gostrich.Sampler  // keeps track of host latency so that we can TODO mark dead
	status        NodeStatus        // mutable field
	flaky	      float64           // what's average latency to be considered flaky in millisecond
	dead          float64           // what's average latency to be considered dead in millisecond
	proberRunning int32             // whether there's a prober started already
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
func NewTransportWithHost(host string) *TransportWithHost {
	return &TransportWithHost{ host, http.Transport{}, gostrich.NewSampler(100), NODE_ALIVE, 1000 * 1000, 9500 * 1000, 0 }
}
/*
* Simple rule that allows filter based on Host/port and resource prefix.
*/
type PrefixRewriteRule struct {
	// transformation rules
	SourceHost           string    // "" matches all
	SourcePathPrefix     string
	ProxiedPathPrefix    string
	ProxiedAttachHeaders map[string][]string

	// clients to use
	Clients              []*TransportWithHost
}

func (p *PrefixRewriteRule) HandlesRequest(r *http.Request) bool {
	return (p.SourceHost == "" || p.SourceHost == r.Host) &&
		strings.HasPrefix(r.URL.Path, p.SourcePathPrefix)
}

func (p *PrefixRewriteRule) TransformRequest(r *http.Request) *TransportWithHost {
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

type float64Slice []float64
func (ns float64Slice) average() float64 {
	sum := 0.0
	for _, v := range ns {
		sum += v
	}
	return sum / float64(len(ns))
}

func requestWithProber(client *TransportWithHost, req *http.Request) (rsp *http.Response, err error) {
	then := time.Now()
	rsp, err = client.transport.RoundTrip(req)
	now :=  time.Now()

	// micro seconds
	latency := (now.Second() - then.Second()) * 1000000 +
		(now.Nanosecond() - then.Nanosecond()) / 1000

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
					requestWithProber(client, req)
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
			client := rule.TransformRequest(r)

			fmt.Print(fmt.Sprintf("And host's transformed to %v\n", r.URL.Host))
			fmt.Print(fmt.Sprintf("And path's transformed to %v\n", r.URL.Path))
			fmt.Print(fmt.Sprintf("And header's transformed to %v\n", r.Header))
			fmt.Print(fmt.Sprintf("%v\n", r))

			rsp, err := requestWithProber(client, r)

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
}
