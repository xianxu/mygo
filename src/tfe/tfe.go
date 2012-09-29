package tfe

/*
 * A simple front end server that proxies request, as in Tfe
 */
import (
	"gostrich"

	"io"
	"net/http"
	"strings"
	"time"
	"log"
)

// bunch of alias to make rule writing more descriptive
type RequestHost string
type RequestPrefix string
type ProxiedPrefix string
type ProxiedHost string
type Retries int
type Timeout time.Duration
type MaxIdleConnsPerHost int

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
type prefixRewriteRule struct {
	// transformation rules
	sourceHost           string // "" matches all
	sourcePathPrefix     string
	proxiedPathPrefix    string
	proxiedAttachHeaders map[string][]string
	//TODO: how to enforce some type checking, we don't want any service, but some HttpService
	service              Service
}

func NewPrefixRule(rh RequestHost,
rp RequestPrefix,
pp ProxiedPrefix,
pah map[string][]string,
s Service) *prefixRewriteRule {
	return &prefixRewriteRule{
		string(rh),
		string(rp),
		string(pp),
		pah,
		s,
	}
}

func (p *prefixRewriteRule) HandlesRequest(r *http.Request) bool {
	return (p.sourceHost == "" || p.sourceHost == r.Host) &&
		strings.HasPrefix(r.URL.Path, p.sourcePathPrefix)
}

func (p *prefixRewriteRule) TransformRequest(r *http.Request) {
	r.URL.Path = p.proxiedPathPrefix + r.URL.Path[len(p.sourcePathPrefix):len(r.URL.Path)]
	r.RequestURI = ""
	for k, v := range p.proxiedAttachHeaders {
		r.Header[k] = v
	}
}

func (p *prefixRewriteRule) TransformResponse(rsp *http.Response) {
	//TODO
	return
}

func (p *prefixRewriteRule) GetService() Service {
	return p.service
}

func (rs *Rules) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, rule := range ([]Rule)(*rs) {
		if rule.HandlesRequest(r) {
			rule.TransformRequest(r)
			s := rule.GetService()
			rawRsp, err := s.Serve(r)
			rsp := rawRsp.(*http.Response)

			if err != nil {
				log.Printf("Error occurred while proxying: %v\n", err.Error())
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
		rsp := rawRsp.(*http.Response)

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
