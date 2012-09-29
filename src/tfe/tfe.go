package tfe

/*
 * A simple front end server that proxies request, as in Tfe
 */
import (
	"gostrich"

	"io"
	"net/http"
	"strings"
	"log"
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
	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request
	TransformRequest(*http.Request)

	// which service to use, this is the actual entity that is capable of handle request
	GetService() Service

	// mutate response
	TransformResponse(*http.Response)
}

/*
* Simple rule implementation that allows filter based on Host/port and resource prefix.
 */
type PrefixRewriteRule struct {
	// transformation rules
	SourceHost           string // "" matches all
	SourcePathPrefix     string
	ProxiedPathPrefix    string
	ProxiedAttachHeaders map[string][]string
	//TODO: how to enforce some type checking, we don't want any service, but some HttpService
	Service              Service
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

func (p *PrefixRewriteRule) GetService() Service {
	return p.Service
}

func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {
			rule.TransformRequest(r)
			s := rule.GetService()
			if s == nil {
				log.Printf("No service defined for rule")
				w.WriteHeader(404)
				return
			}

			rawRsp, err := s.Serve(r)

			if err != nil {
				log.Printf("Error occurred while proxying: %v\n", err.Error())
				w.WriteHeader(503)
				return
			}

			rsp := rawRsp.(*http.Response)
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
				// TODO, shouldn't happen, but it does happen :S
				log.Printf("err while piping bytes: %v\n", err)
			}

			return
		}
	}

	w.WriteHeader(404)
	return
}

//TODO: stats collection's pushed into "Cluster", need to figure out what to do there.
func HttpStats(stats gostrich.Stats) func(interface{}, interface{}, error, int) {
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

	return func(rawReq interface{}, rawRsp interface{}, err error, micro int) {
        /*req := rawReq.(*http.Request)*/
		counterReq.Incr(1)
		if err != nil {
			counterFail.Incr(1)
		} else {
			counterSucc.Incr(1)
			rsp := rawRsp.(*http.Response)
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
