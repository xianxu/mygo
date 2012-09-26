package tfe

import (
	"gostrich"

	"fmt"
	"net/http"
	"strings"
	"time"
	"net/url"
	"io"
	"sync/atomic"
)

type Rules []Rule

/*
* A Tfe is basically set of rules handled by it.
*/
type Tfe struct {
	BindingToRules map[string]Rules
}

/*
* A routing rule means three things:
*   - to determine whether a rule can be applied to the incoming request
*   - to transform request before forwarding
*   - to transform response before sending upstream
*/
type Rule interface {
	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request and return a specific downstream client to connect to
	TransformRequest(*http.Request) *TransportWithHost

	// mutate response
	TransformResponse(*http.Response)
}

type NodeStatus int32
const (
	NODE_ALIVE = NodeStatus(0)  // node will be queried normally
	NODE_FLAKY = NodeStatus(1)	// node seems flaky, we will send lower traffic
	NODE_DEAD  = NodeStatus(4)	// node seems dead, we will send even lower traffic
	//TODO permanently dead? avoid herding
)

/*
* Keep tracks of host, connection pool for each host and latency history. Will do meaningful
* balancing later but for now we'd just round robin.
*/
type TransportWithHost struct {
	hostPort  string
	transport http.Transport
	latencies gostrich.Sampler  // keeps track of host latency so that we can TODO mark dead
	status    NodeStatus
	flaky	  float64           // what's average latency to be considered flaky in millisecond
	dead      float64           // what's average latency to be considered dead in millisecond
}

//TODO: we will need to tweak numbers, only keeping track of 10 seems too reactive.
// Make a new host with construct to keep track of its health. Internally we keep last 10 call's
// latency and treat failures as taking 10 seconds. This means a single error will make average
// latency greater than 1 second. If we want to react to 5 errors out of last 10 requests, we'd
// set flaky to 5000 and dead to 10000, probably.
func NewTransportWithHost(host string) *TransportWithHost {
	return &TransportWithHost{ host, http.Transport{}, gostrich.NewSampler(10), NODE_ALIVE, 5000 * 1000, 1000 * 1000 }
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

func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {
			fmt.Print("Routes detected\n")
			client := rule.TransformRequest(r)

			fmt.Print(fmt.Sprintf("And host's transformed to %v\n", r.URL.Host))
			fmt.Print(fmt.Sprintf("And path's transformed to %v\n", r.URL.Path))
			fmt.Print(fmt.Sprintf("And header's transformed to %v\n", r.Header))
			fmt.Print(fmt.Sprintf("%v\n", r))

			then := time.Now()

			rsp, err := client.transport.RoundTrip(r)

			now :=  time.Now()

			// micro seconds
			latency := (now.Second() - then.Second()) * 1000000 +
				(now.Nanosecond() - then.Nanosecond()) / 1000

			if err != nil {
				latency = 10 * 1000000
			}

			client.latencies.Observe(float64(latency))
			avg := float64Slice(client.latencies.Sampled()).average()

			switch {
			case avg > client.dead:
				atomic.StoreInt32((*int32)(&client.status), int32(NODE_DEAD))
			case avg > client.flaky:
				atomic.StoreInt32((*int32)(&client.status), int32(NODE_FLAKY))
			default:
				atomic.StoreInt32((*int32)(&client.status), int32(NODE_ALIVE))
			}
			fmt.Printf("Client status is: %v\n", client.status)

			fmt.Printf("Latencies avg: %v\n", avg)
			fmt.Printf("%v", client.latencies.Sampled())

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
