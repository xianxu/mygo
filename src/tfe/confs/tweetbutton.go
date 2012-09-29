package confs

import (
	"gostrich"
	"tfe"
	"time"
	"log"
	"net/http"
)

func init() {
	ok := tfe.AddRules("tweetbutton-smf1", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				tfe.NewPrefixRule(
					tfe.RequestHost(""),
					tfe.RequestPrefix("/1/urls/"),
					tfe.ProxiedPrefix("/1/urls/"),
					map[string][]string{
						"True-Client-Ip": []string{"127.0.0.1"},
					},
					tfe.NewCluster(
						[]*tfe.ServiceWithHistory{
							tfe.NewServiceWithHistory(
								tfe.NewServiceWithTimeout(
									&tfe.HttpService { &http.Transport {}, "smf1-aea-35-sr2:8000" },
									time.Second),
								"smf1-aea-35-sr2:8000",
								tfe.ServiceReporter(tfe.HttpStats(gostrich.StatsSingleton().Scoped("tweetbutton").Scoped("smf1-aea-35-sr2:8000")))),
						},
						"tweetbutton",
						2,
						tfe.ServiceReporter(tfe.HttpStats(gostrich.StatsSingleton().Scoped("tweetbutton")))),
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
		log.Println("Rule set named tweetbutton-smf1 already exists")
	}
}
