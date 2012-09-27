package main

import (
	"gostrich"
	"tfe"

	"fmt"
	"flag"
	"net/http"
	"time"
)

var (
	theTfe *tfe.Tfe
)

//TODO:
//      - move to config file?
//      - large response
//      - post request
//      - gz support
//      - chunked encoding
func init() {
	theTfe = &tfe.Tfe {
		map[string]tfe.Rules {
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
		},
	}
	fmt.Println("init done.")
}

func main() {
	flag.Parse()
	for binding, rules := range theTfe.BindingToRules {
		server := http.Server {
			binding,
			&rules,
			18 * time.Second,
			10 * time.Second,
			0,
			nil,
		}
		go server.ListenAndServe()
	}
	gostrich.StartToLive()
}
