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
	conf = flag.String("rules", "empty", "rules to run, comma seperated")
	numcpu = flag.Int("numcpu", 1, "number of cpu to use")
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
)

//TODO tests:
//      - large response
//      - post request
//      - gz support
//      - chunked encoding

func main() {
	flag.Parse()
	runtime.GOMAXPROCS(*numcpu)

	if *cpuprofile != "" {
		log.Println("Enabling profiling")
        f, err := os.Create(*cpuprofile)
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
			10 * time.Second,
			10 * time.Second,
			0,
			nil, // SSL TODO
		}
		go server.ListenAndServe()
	}
	gostrich.StartToLive()
	log.Println("Stopped TFE")
}
