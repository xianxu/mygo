package tfe

import (
	"time"
	"gostrich"
	"net/http"
)

type StaticHttpCluster struct {
	Name string
	Hosts []string
	Timeout time.Duration
	Retries int
}
func CreateStaticHttpCluster(config StaticHttpCluster) *cluster {
	services := make([]*ServiceWithHistory, len(config.Hosts))
	for i, h := range config.Hosts {
		httpService := &HttpService { &http.Transport {}, h }
		withTimeout := NewServiceWithTimeout(httpService, config.Timeout)
		services[i] = NewServiceWithHistory( withTimeout, h, ServiceReporter(HttpStats(gostrich.StatsSingleton().Scoped(config.Name).Scoped(h))))
	}
	return NewCluster(
		services,
		config.Name,
		config.Retries + 1,
		ServiceReporter(HttpStats(gostrich.StatsSingleton().Scoped(config.Name))))

}
