package confs

import (
	"tfe"
	"time"
	"log"
	"gostrich"
)

func init() {
	ok := tfe.AddRules("test", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				&tfe.PrefixRewriteRule{
					Name:              "tco",
					SourcePathPrefix:  "/tco/",
					ProxiedPathPrefix: "/",
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster{
							Name: "tco",
							Hosts: []string{
								"t.co",
							},
							Timeout:   1 * time.Second,
							Retries:   1,
							ProberReq: tfe.ProberReqLastFail,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.StatsSingleton().Scoped("tfe-tco")),
				},
				&tfe.PrefixRewriteRule{
					Name:              "tweetbutton",
					SourcePathPrefix:  "/urls-real/",
					ProxiedPathPrefix: "/1/urls/",
					ProxiedAttachHeaders: map[string][]string{
						"True-Client-Ip": []string{"127.0.0.1"},
					},
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster{
							Name: "tbapi",
							Hosts: []string{
								"urls-real.api.twitter.com",
							},
							Timeout:   1 * time.Second,
							Retries:   1,
							ProberReq: tfe.ProberReqLastFail,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.StatsSingleton().Scoped("tfe-tbapi")),
				},
			},
		}
	})

	if !ok {
		log.Println("Rule set named test already exists")
	}
}
