package tfe

/*
 * A simple front end server that proxies request, as in Tfe
 */
import (
	"io"
	"net/http"
	"strings"
	"log"
	"strconv"
	"time"
	"rpcx"
)

var (
	contentLength0 = []string{"0"}
)

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
* It also provides a way to report stats on the service.
 */
type Rule interface {
	// name of this rule
	GetName() string

	// which service to use, this is the actual entity that is capable of handle http.Request
	GetService() rpcx.Service

	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request
	TransformRequest(*http.Request)

	// mutate response
	TransformResponse(*http.Response)

	// get a reporter to report overall response stats
	GetServiceReporter() rpcx.ServiceReporter
}

/*
* Simple rule implementation that allows filter based on Host/port and resource prefix.
 */
type PrefixRewriteRule struct {
	Name string
	// transformation rules
	SourceHost           string // "" matches all
	SourcePathPrefix     string
	ProxiedPathPrefix    string
	ProxiedAttachHeaders map[string][]string
	//TODO: how to enforce some type checking, we don't want any service, but some HttpService
	Service  rpcx.Service
	Reporter rpcx.ServiceReporter
}

func (p *PrefixRewriteRule) HandlesRequest(r *http.Request) bool {
	return (p.SourceHost == "" || p.SourceHost == r.Host) &&
		strings.HasPrefix(r.URL.Path, p.SourcePathPrefix)
}

func (p *PrefixRewriteRule) TransformRequest(r *http.Request) {
	r.URL.Path = p.ProxiedPathPrefix + r.URL.Path[len(p.SourcePathPrefix):len(r.URL.Path)]
	r.RequestURI = ""
	if p.ProxiedAttachHeaders != nil {
		for k, v := range p.ProxiedAttachHeaders {
			r.Header[k] = v
		}
	}
}

func (p *PrefixRewriteRule) TransformResponse(rsp *http.Response) {
	//TODO
	return
}

func (p *PrefixRewriteRule) GetService() rpcx.Service {
	return p.Service
}

func (p *PrefixRewriteRule) GetName() string {
	return p.Name
}

func (p *PrefixRewriteRule) GetServiceReporter() rpcx.ServiceReporter {
	return p.Reporter
}

func report(reporter rpcx.ServiceReporter, req *http.Request, rsp interface{}, err error, l int) {
	if reporter != nil {
		reporter.Report(req, rsp, err, l)
	}
}

// Tfe HTTP serving endpoint.
func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	then := time.Now()
	headers := w.Header()
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {
			ruleName := rule.GetName()
			rule.TransformRequest(r)
			s := rule.GetService()
			reporter := rule.GetServiceReporter()

			if s == nil {
				log.Printf("No service defined for rule %v\n", ruleName)
				headers["Content-Length"] = contentLength0
				w.WriteHeader(404)
				report(reporter, r, &SimpleResponseForStat{404, 0}, nil, rpcx.MicroTilNow(then))
				return
			}

			// replace r.Body with CachedReader so if we need to retry, we can.
			var err error
			if r.Body != nil {
				if r.Body, err = NewCachedReader(r.Body); err != nil {
					// if we can't read request body, just fail
					log.Printf("Error occurred while reading request body for rule %v: %v\n",
						ruleName, err.Error())
					headers["Content-Length"] = contentLength0
					w.WriteHeader(503)
					report(reporter, r, &SimpleResponseForStat{503, 0}, nil, rpcx.MicroTilNow(then))
					return
				}
			}

			var rsp http.Response

			// TODO: better interface, timeout's not used here, it's specified when underlying
			// HttpService is created.
			err = s.Serve(r, &rsp, time.Second)

			if err != nil {
				log.Printf("Error occurred while proxying for rule %v: %v\n", ruleName, err.Error())
				headers["Content-Length"] = contentLength0
				w.WriteHeader(503)
				report(reporter, r, &SimpleResponseForStat{503, 0}, nil, rpcx.MicroTilNow(then))
				return
			}

			rule.TransformResponse(&rsp)

			for k, v := range rsp.Header {
				headers[k] = v
			}
			if body, ok := rsp.Body.(*CachedReader); ok {
				// output content length if we know it
				headers["Content-Length"] = []string{strconv.Itoa(len(body.Bytes))}
			}
			w.WriteHeader(rsp.StatusCode)

			if rsp.StatusCode >= 300 && rsp.StatusCode < 400 {
				// if redirects
				report(reporter, r, &rsp, nil, rpcx.MicroTilNow(then))
				return
			}

			// all other with body
			_, err = io.Copy(w, rsp.Body)

			// log error while copying
			if err != nil {
				// TODO, shouldn't happen, but it does happen :S
				// TODO, stats report
				log.Printf("err while piping bytes for rule %v: %v\n", ruleName, err)
			}

			report(reporter, r, &rsp, nil, rpcx.MicroTilNow(then))
			return
		}
	}

	headers["Content-Length"] = contentLength0
	w.WriteHeader(404)
	return
}
