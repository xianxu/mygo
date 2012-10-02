package tfe

import (
	"net/http"
	"errors"
	"log"
)

var (
	errReqType = errors.New("HttpService: expect *http.Request as request object.")
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

func (h *HttpService) Serve(req interface{}) (rsp interface{}, err error) {
	var httpReq *http.Request
	var ok bool
	rsp = nil

	if httpReq, ok = req.(*http.Request); !ok {
		err = errReqType
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

	httpRsp, err := h.RoundTrip(httpReq)
	rsp = httpRsp

	// cache response body, in order to report stats on size and output Content-Length header.
	// Without, upstream can only do chunked encoding and without reporting size.
	if h.CacheResponseBody && httpRsp != nil && httpRsp.Body != nil {
		if httpRsp.Body, err = NewCachedReader(httpRsp.Body); err != nil {
			// if we can't read request body, just fail
			log.Printf("Error occurred while reading response body: %v\n", err.Error())
		}
	}

	return
}
