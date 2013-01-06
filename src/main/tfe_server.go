package main

import (
	"gostrich"
	"tfe"
	_ "tfe/confs"

	"flag"
	"log"
	"net/http"
	"time"
)

var (
	conf         = flag.String("rules", "empty", "Rules to run, comma seperated")
	readTimeout  = flag.Duration("read_timeout", 10*time.Second, "Read timeout")
	writeTimeout = flag.Duration("write_timeout", 10*time.Second, "Write timeout")
)

//TODO tests:
//      - large response
//      - post request
//      - gz support
//      - chunked encoding

func main() {
	flag.Parse()
	portOffset := *gostrich.PortOffset

	log.Printf("Starting TFE with rule: %v, read timeout: %v, write timeout: %v, with port_offset: %v",
		*conf, *readTimeout, *writeTimeout, portOffset)
	theTfe := &tfe.Tfe{tfe.GetRules(*conf, portOffset)()}
	for binding, rules := range theTfe.BindingToRules {
		server := http.Server{
			binding,
			&rules,
			*readTimeout,
			*writeTimeout,
			0,
			nil, // SSL TODO
		}
		go server.ListenAndServe()
	}
	gostrich.StartToLive(nil)
}
