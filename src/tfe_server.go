package main

import (
	"gostrich"
	"tfe"
	_ "tfe/confs"

	"flag"
	"net/http"
	"time"
	"code.google.com/p/log4go"
)

var (
	conf = flag.String("rules", "empty", "rules to run, comma seperated")
)

//TODO tests:
//      - large response
//      - post request
//      - gz support
//      - chunked encoding

func main() {
	flag.Parse()
	log4go.Info("Starting TFE")
	theTfe := &tfe.Tfe{tfe.GetRules(*conf)()}
	for binding, rules := range theTfe.BindingToRules {
		server := http.Server{
			binding,
			&rules,
			10 * time.Second,
			10 * time.Second,
			0,
			nil, // SSL TODO
		}
		go server.ListenAndServe()
	}
	gostrich.StartToLive()
	log4go.Info("Stopped TFE")
}
