package confs

import (
	"tfe"
	"time"
	"log"
	"gostrich"
	"rpcx"
)

func init() {
	ok := tfe.AddRules("tweetbutton-tunnels", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				&tfe.PrefixRewriteRule{
					Name:              "tweetbutton-tunnels",
					SourcePathPrefix:  "/1/urls/",
					ProxiedPathPrefix: "/1/urls/",
					ProxiedAttachHeaders: map[string][]string{
						"True-Client-Ip": []string{"127.0.0.1"},
					},
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster{
							Name: "tweetbutton-tunnels",
							Hosts: []string{
								"localhost:8000", // self
								"localhost:8001",
								"localhost:8002",
								"localhost:8003",
								"localhost:8004",
								"localhost:8005",
								"localhost:8006",
								"localhost:8007",
								"localhost:8008",
							},
							Timeout:   2 * time.Second,
							Retries:   1,
							ProberReq: rpcx.ProberReqLastFail,
						}),
					Reporter: tfe.NewHttpStatsReporter(gostrich.AdminServer().GetStats().Scoped("tweetbutton-tunneled")),
				},
			},
		}
	})

	if !ok {
		log.Println("Rule set named tweetbutton-tunnels already exists")
	}
}
