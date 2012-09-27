package confs

import (
	"tfe"
	"time"
	"code.google.com/p/log4go"
)

func init() {
	ok := tfe.AddRules("test", func() map[string]tfe.Rules {
		return map[string]tfe.Rules {
			":8887": tfe.Rules {
				tfe.NewPrefixRule(
					tfe.RuleName("tweetbutton-prod-hack"),
					tfe.RequestHost(""),
					tfe.RequestPrefix("/1/urls/"),
					tfe.ProxiedPrefix("/1/urls/"),
					map[string][]string {
						"True-Client-Ip": []string{"127.0.0.1",},
					},
					[]*tfe.TransportWithHost {
						tfe.NewTransportWithHost("localhost:8000", tfe.MaxIdleConnsPerHost(10)),
					},
					tfe.Retries(1),
					tfe.Timeout(3 * time.Second),
				),
			},
			":8888": tfe.Rules {
				tfe.NewPrefixRule(
					tfe.RuleName("tweetbutton-hack"),
					tfe.RequestHost(""),
					tfe.RequestPrefix("/urls-real/"),
					tfe.ProxiedPrefix("/1/urls/"),
					map[string][]string {
						"True-Client-Ip": []string{"127.0.0.1",},
					},
					[]*tfe.TransportWithHost {
						tfe.NewTransportWithHost("urls-real.api.twitter.com", tfe.MaxIdleConnsPerHost(10)),
					},
					tfe.Retries(1),
					tfe.Timeout(3* time.Second),
				),
				tfe.NewPrefixRule(
					tfe.RuleName("tco-hack"),
					tfe.RequestHost(""),
					tfe.RequestPrefix("/tco/"),
					tfe.ProxiedPrefix("/"),
					map[string][]string {
					},
					[]*tfe.TransportWithHost {
						tfe.NewTransportWithHost("t.co", tfe.MaxIdleConnsPerHost(10)),
					},
					tfe.Retries(1),
					tfe.Timeout(3 * time.Second),
				),
			},
		}
	})

	if !ok {
		log4go.Warn("Rule set named test already exists")
	}
}
