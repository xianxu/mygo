package tfe

import (
	"gostrich"

	"fmt"
	"net/http"
	"strings"
	"time"
	"net/url"
	"io"
)

type Rules []Rule

/*
* A Tfe is basically set of rules handled by it.
*/
type Tfe struct {
	BindingToRules map[string]Rules
}

//TODO: probably make immutable
type Rule interface {
	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request and return a specific downstream client to connect to
	TransformRequest(*http.Request) *TransportWithHost

	// mutate response
	TransformResponse(*http.Response)
}

/*
* Keep tracks of host, connection pool for each host and latency history. Will do meaningful
* balancing later but for now we'd just round robin.
*/
type TransportWithHost struct {
	hostPort  string
	transport http.Transport
	latencies gostrich.Sampler  // keeps track of host latency, TODO: some type of balancing
}

func NewTransportWithHost(host string) *TransportWithHost {
	return &TransportWithHost{ host, http.Transport{}, gostrich.NewSampler(10) }
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
	client := p.Clients[0] //TODO: fancier load balancing.

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
				latency *= 10  // treat error as 10x latency.
			}

			client.latencies.Observe(float64(latency))

			fmt.Printf("Latencies avg: %v\n", float64Slice(client.latencies.Sampled()).average())
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
