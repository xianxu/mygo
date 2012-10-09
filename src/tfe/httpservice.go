package tfe

import (
	"net/http"
	"errors"
	"log"
	"gostrich"
	"time"
)

var (
	errReqType = errors.New("HttpService: expect *http.Request as request object.")
	errRspType = errors.New("HttpService: expect *http.Response as response object.")
)

//TODO: implement HostConnectionLimit in Finagle world? seems both good and bad. There's no
//much need assuming excess connections alone will not affect downstream service.
type HttpService struct {
	http.RoundTripper
	// this http service will rewrite request to this host port
	HostPort string
	// whether to cache resonse body
	CacheResponseBody bool
}

func (h *HttpService) Serve(req interface{}, rsp interface{}, timeout time.Duration) (err error) {
	var httpReq *http.Request
	var httpRsp *http.Response
	var ok bool

	if httpReq, ok = req.(*http.Request); !ok {
		err = errReqType
		return
	}

	if httpRsp, ok = rsp.(*http.Response); !ok {
		err = errRspType
		return
	}

	// if hostPort's not set as we need, update it. this happens if we load balance to another
	// host on retry.
	if httpReq.URL.Host != h.HostPort {
		httpReq.URL.Scheme = "http" // TODO: hack?
		httpReq.URL.Host = h.HostPort
		httpReq.Host = h.HostPort
	}

	var cr *CachedReader
	if cr, ok = httpReq.Body.(*CachedReader); ok {
		// if it's a cached reader, let's reset it, this happens during retry.
		cr.Reset()
	}

	httpRsp1, err := h.RoundTrip(httpReq)
	// TODO: does this work to make a shallow copy?
	if httpRsp1 != nil {
		*httpRsp = *httpRsp1
	}

	// cache response body, in order to report stats on size and output Content-Length header.
	// Without, upstream can only do chunked encoding and without reporting size.
	if h.CacheResponseBody && httpRsp1 != nil && httpRsp.Body != nil {
		if httpRsp.Body, err = NewCachedReader(httpRsp.Body); err != nil {
			// if we can't read request body, just fail
			log.Printf("Error occurred while reading response body: %v\n", err.Error())
		}
	}

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

func (h *HttpStatsReporter) Report(rawReq interface{}, rawRsp interface{}, err error, micro int64) {
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

		h.sizeStat.Observe(int64(size))
		switch {
		case code >= 100 && code < 200:
			h.counter1xx.Incr(1)
			h.size1xx.Observe(int64(size))
		case code >= 200 && code < 300:
			h.counter2xx.Incr(1)
			h.size2xx.Observe(int64(size))
		case code >= 300 && code < 400:
			h.counter3xx.Incr(1)
			h.size3xx.Observe(int64(size))
		case code >= 400 && code < 500:
			h.counter4xx.Incr(1)
			h.size4xx.Observe(int64(size))
		case code >= 500 && code < 600:
			h.counter5xx.Incr(1)
			h.size5xx.Observe(int64(size))
		default:
			h.counterRst.Incr(1)
			h.sizeRst.Observe(int64(size))
		}
	}
}
