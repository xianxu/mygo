package main

import (
	"gostrich"
	"tfe"

	"flag"
	"net/http"
	"time"
)

var (
	theTfe *tfe.Tfe
)

//TODO: move to config file
func init() {
	theTfe = &tfe.Tfe {
		map[string]tfe.Rules {
			":8888": tfe.Rules {
				&tfe.PrefixRewriteRule {
					"",
					"/urls/",
					"/1/urls/",
					map[string][]string {
						"True-Client-Ip": []string{"127.0.0.1",},
					},
					[]*tfe.TransportWithHost {
						tfe.NewTransportWithHost("localhost:8000"),
					},
				},
				&tfe.PrefixRewriteRule {
					"",
					"/tco/",
					"/",
					map[string][]string {
					},
					[]*tfe.TransportWithHost {
						tfe.NewTransportWithHost("t.co"),
					},
				},
			},
		},
	}
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
