package main

import (
	"gostrich"
	"tfe"
	_ "tfe/confs"

	"flag"
	"net/http"
	"runtime"
	"time"
	"log"
	"os"

	"runtime/pprof"
	_ "net/http/pprof"
)

var (
	conf         = flag.String("rules", "empty", "rules to run, comma seperated")
	numCPU       = flag.Int("numcpu", 1, "number of cpu to use")
	cpuProfile   = flag.String("cpuprofile", "", "write cpu profile to file")
	readTimeout  = flag.Duration("read_timeout", 10*time.Second, "read timeout")
	writeTimeout = flag.Duration("write_timeout", 10*time.Second, "read timeout")
)

//TODO tests:
//      - large response
//      - post request
//      - gz support
//      - chunked encoding

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*numCPU)

	if *cpuProfile != "" {
		log.Println("Enabling profiling")
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.Println("Starting TFE")
	theTfe := &tfe.Tfe{tfe.GetRules(*conf)()}
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
	gostrich.StartToLive()
	log.Println("Stopped TFE")
}
