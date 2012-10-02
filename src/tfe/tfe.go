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
	"strconv"
	"time"
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
	GetService() Service

	// whether this rule handles this request
	HandlesRequest(*http.Request) bool

	// mutate request
	TransformRequest(*http.Request)

	// mutate response
	TransformResponse(*http.Response)

	// get a reporter to report overall response stats
	GetServiceReporter() ServiceReporter
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
	Service  Service
	Reporter ServiceReporter
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

func (p *PrefixRewriteRule) GetName() string {
	return p.Name
}

func (p *PrefixRewriteRule) GetServiceReporter() ServiceReporter {
	return p.Reporter
}

func report(reporter ServiceReporter, req *http.Request, rsp interface{}, err error, l int) {
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
				report(reporter, r, &SimpleResponseForStat{404, 0}, nil, microTilNow(then))
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
					report(reporter, r, &SimpleResponseForStat{503, 0}, nil, microTilNow(then))
					return
				}
			}

			rawRsp, err := s.Serve(r)

			if err != nil {
				log.Printf("Error occurred while proxying for rule %v: %v\n", ruleName, err.Error())
				headers["Content-Length"] = contentLength0
				w.WriteHeader(503)
				report(reporter, r, &SimpleResponseForStat{503, 0}, nil, microTilNow(then))
				return
			}

			rsp := rawRsp.(*http.Response)
			rule.TransformResponse(rsp)

			for k, v := range rsp.Header {
				headers[k] = v
			}
			if body, ok := rsp.Body.(*CachedReader); ok {
				// output content length if we know it
				headers["Content-Length"] = []string{strconv.Itoa(len(body.Bytes))}
			}
			w.WriteHeader(rsp.StatusCode)

			if rsp != nil && rsp.StatusCode >= 300 && rsp.StatusCode < 400 {
				// if redirects
				report(reporter, r, rsp, nil, microTilNow(then))
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

			report(reporter, r, rsp, nil, microTilNow(then))
			return
		}
	}

	headers["Content-Length"] = contentLength0
	w.WriteHeader(404)
	return
}

/*
* Simple struct used to carry enough information for stats reporting purpose
 */
type SimpleResponseForStat struct {
	StatusCode    int
	ContentLength int
}

type HttpStatsReporter struct {
	counterReq, counterSucc, counterFail, counterRspNil, counterRspTypeErr gostrich.Counter
	counter1xx, counter2xx, counter3xx, counter4xx, counter5xx, counterRst gostrich.Counter
	reqLatencyStat, sizeStat                                               gostrich.IntSampler
	size1xx, size2xx, size3xx, size4xx, size5xx, sizeRst                   gostrich.IntSampler
}

func NewHttpStatsReporter(stats gostrich.Stats) *HttpStatsReporter {
	return &HttpStatsReporter{
		counterReq:     stats.Counter("req"),
		counterSucc:    stats.Counter("req/success"),
		counterFail:    stats.Counter("req/fail"),
		reqLatencyStat: stats.Statistics("req/latency"),

		sizeStat:          stats.Statistics("rsp/size"),
		counterRspNil:     stats.Counter("rsp/nil"),
		counterRspTypeErr: stats.Counter("rsp/type_err"),
		counter1xx:        stats.Counter("rsp/1xx"),
		size1xx:           stats.Statistics("rsp_size/1xx"),
		counter2xx:        stats.Counter("rsp/2xx"),
		size2xx:           stats.Statistics("rsp_size/2xx"),
		counter3xx:        stats.Counter("rsp/3xx"),
		size3xx:           stats.Statistics("rsp_size/3xx"),
		counter4xx:        stats.Counter("rsp/4xx"),
		size4xx:           stats.Statistics("rsp_size/4xx"),
		counter5xx:        stats.Counter("rsp/5xx"),
		size5xx:           stats.Statistics("rsp_size/5xx"),
		counterRst:        stats.Counter("rsp/rst"),
		sizeRst:           stats.Statistics("rsp_size/rst"),
	}
}

func (h *HttpStatsReporter) Report(rawReq interface{}, rawRsp interface{}, err error, micro int) {
	/*req := rawReq.(*http.Request)*/
	h.reqLatencyStat.Observe(micro)
	h.counterReq.Incr(1)

	if err != nil {
		h.counterFail.Incr(1)
	} else {
		h.counterSucc.Incr(1)

		var code, size int

		if rawRsp == nil {
			h.counterRspNil.Incr(1)
			log.Printf("Response passed to HttpStatsReporter is nil\n")
			return
		} else if rsp, ok := rawRsp.(*http.Response); ok {
			code = rsp.StatusCode
			// if cached, use cached size, otherwise rely on ContentLength, which is not reliable.
			if body, ok := rsp.Body.(*CachedReader); ok {
				size = len(body.Bytes)
			} else {
				size = int(rsp.ContentLength)
			}
		} else if rsp, ok := rawRsp.(*SimpleResponseForStat); ok {
			code = rsp.StatusCode
			size = rsp.ContentLength
		} else {
			h.counterRspTypeErr.Incr(1)
			log.Printf("Response passed to HttpStatsReporter is not valid\n")
			return
		}

		h.sizeStat.Observe(size)
		switch {
		case code >= 100 && code < 200:
			h.counter1xx.Incr(1)
			h.size1xx.Observe(size)
		case code >= 200 && code < 300:
			h.counter2xx.Incr(1)
			h.size2xx.Observe(size)
		case code >= 300 && code < 400:
			h.counter3xx.Incr(1)
			h.size3xx.Observe(size)
		case code >= 400 && code < 500:
			h.counter4xx.Incr(1)
			h.size4xx.Observe(size)
		case code >= 500 && code < 600:
			h.counter5xx.Incr(1)
			h.size5xx.Observe(size)
		default:
			h.counterRst.Incr(1)
			h.sizeRst.Observe(size)
		}
	}
}
