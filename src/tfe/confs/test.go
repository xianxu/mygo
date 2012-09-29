package confs

import (
	"tfe"
	"time"
	"log"
)

func init() {
	ok := tfe.AddRules("test", func() map[string]tfe.Rules {
		return map[string]tfe.Rules{
			":8888": tfe.Rules{
				&tfe.PrefixRewriteRule{
					SourcePathPrefix: "/tco/",
					ProxiedPathPrefix: "/",
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster {
							Name: "tco",
							Hosts: []string {
								"t.co",
							},
							Timeout: 1 * time.Second,
							Retries: 1,
						}),
					},
				&tfe.PrefixRewriteRule{
					SourcePathPrefix: "/urls-real/",
					ProxiedPathPrefix: "/1/urls/",
					ProxiedAttachHeaders: map[string][]string {
						"True-Client-Ip": []string{"127.0.0.1"},
					},
					Service: tfe.CreateStaticHttpCluster(
						tfe.StaticHttpCluster {
							Name: "tbapi",
							Hosts: []string {
								"urls-real.api.twitter.com",
							},
							Timeout: 1 * time.Second,
							Retries: 1,
						}),
					},
			},
		}
	})

	if !ok {
		log.Println("Rule set named test already exists")
	}
}
