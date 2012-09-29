package confs

import (
	"gostrich"
	"tfe"
	"time"
	"log"
	"net/http"
)

func init() {
	ok := tfe.AddRules("test", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				tfe.NewPrefixRule(
					tfe.RequestHost(""),
					tfe.RequestPrefix("/tco/"),
					tfe.ProxiedPrefix("/"),
					map[string][]string{},
					tfe.NewCluster(
						[]*tfe.ServiceWithHistory{
							tfe.NewServiceWithHistory(
								tfe.NewServiceWithTimeout(
									&tfe.HttpService { &http.Transport {}, "t.co" },
									time.Second),
								"t.co",
								tfe.ServiceReporter(tfe.HttpStats(gostrich.StatsSingleton().Scoped("tco").Scoped("t.co")))),
						},
						"tco",
						2,
						tfe.ServiceReporter(tfe.HttpStats(gostrich.StatsSingleton().Scoped("tco")))),
					/*[]*tfe.TransportWithHost{*/
						/*tfe.NewTransportWithHost("smf1-aea-35-sr2:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-adz-03-sr3:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-adj-27-sr4:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-afo-35-sr4:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-adz-19-sr2:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-adb-23-sr3:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-adz-27-sr1:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-afe-15-sr3:8000", tfe.MaxIdleConnsPerHost(10)),*/
						/*tfe.NewTransportWithHost("smf1-aer-19-sr4:8000", tfe.MaxIdleConnsPerHost(10)),*/
					/*},*/
				),
			},
		}
	})

	if !ok {
		log.Println("Rule set named test already exists")
	}
}
