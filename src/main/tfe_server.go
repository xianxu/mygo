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
	conf         = flag.String("rules", "empty", "Rules to run, comma seperated")
	numCPU       = flag.Int("numcpu", 1, "Number of cpu to use. Use 0 to use all CPU")
	cpuProfile   = flag.String("cpuprofile", "", "Write cpu profile to file")
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
	ncpu := *numCPU
	if ncpu == 0 {
		ncpu = runtime.NumCPU()
	}
	runtime.GOMAXPROCS(ncpu)

	if *cpuProfile != "" {
		log.Println("Enabling profiling")
		f, err := os.Create(*cpuProfile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	log.Printf("Starting TFE with rule: %v, cpu: %v, read timeout: %v, write timeout: %v",
		*conf, ncpu, *readTimeout, *writeTimeout)
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
