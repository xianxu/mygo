package confs

import (
	"tfe"
	"time"
	"log"
	"gostrich"
	"rpcx"
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
							ProberReq: rpcx.ProberReqLastFail,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped("tfe-tco")),
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
							ProberReq: rpcx.ProberReqLastFail,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped("tfe-tbapi")),
				},
				&tfe.PrefixRewriteRule{
					Name:              "dead",
					SourcePathPrefix:  "/dead/",
					ProxiedPathPrefix: "/dead/",
				},
			},
		}
	})

	if !ok {
		log.Println("Rule set named test already exists")
	}
}
