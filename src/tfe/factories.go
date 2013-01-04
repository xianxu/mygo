package tfe

import (
	"time"
	"gostrich"
	"net/http"
	"rpcx"
)

type StaticHttpCluster struct {
	Name              string
	Hosts             []string
	Timeout           time.Duration
	Retries           int
	ProberReq         interface{}
	CacheResponseBody bool

	// http.Transport config
	DisableKeepAlives   bool
	DisableCompression  bool
	MaxIdleConnsPerHost int
}

func CreateStaticHttpCluster(config StaticHttpCluster) *rpcx.Cluster {
	services := make([]*rpcx.Supervisor, len(config.Hosts))
	for i, h := range config.Hosts {
		httpService := &HttpService{&http.Transport{}, h, config.CacheResponseBody}
		withTimeout := &rpcx.ServiceWithTimeout{httpService, config.Timeout}
		services[i] = rpcx.NewSupervisor(
			h,
			withTimeout,
			NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped(config.Name).Scoped(h)),
			config.ProberReq,
			nil, // no need to recreate client since http.Transport does those alrady
		)
	}
	return &rpcx.Cluster{
		Name:     config.Name,
		Services: services,
		Retries:  config.Retries,
		Reporter: NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped(config.Name)),
	}

}
