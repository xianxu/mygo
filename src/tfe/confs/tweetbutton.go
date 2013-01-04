package confs

import (
	"tfe"
	"time"
	"log"
	"gostrich"
	"rpcx"
)

func init() {
	ok := tfe.AddRules("tweetbutton-smf1", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				&tfe.PrefixRewriteRule{
					Name:              "tweetbutton-smf1-plaintext",
					SourcePathPrefix:  "/1/urls/",
					ProxiedPathPrefix: "/1/urls/",
					ProxiedAttachHeaders: map[string][]string{
						"True-Client-Ip": []string{"127.0.0.1"},
					},
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster{
							Name: "tweetbutton",
							Hosts: []string{
								/*"smf1-aea-35-sr2:8000",*/
								/*"smf1-adz-03-sr3:8000",*/
								"smf1-adj-27-sr4:8000",
								"smf1-afo-35-sr4:8000",
								"smf1-adz-19-sr2:8000",
								"smf1-adb-23-sr3:8000",
								"smf1-adz-27-sr1:8000",
								"smf1-afe-15-sr3:8000",
								"smf1-aer-19-sr4:8000",
							},
							Timeout:             2 * time.Second,
							Retries:             1,
							ProberReq:           rpcx.ProberReqLastFail,
							CacheResponseBody:   true,
							MaxIdleConnsPerHost: 20,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped("tfe-tbapi-smf1-plaintext")),
				},
			},
		}
	})

	if !ok {
		log.Println("Rule set named tweetbutton-smf1 already exists")
	}
}
