package gostrich

/*
 * Ostrich in go, so that we can play go in Twitter's infrastructure.
 */
import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"net/http"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"log"
)

// expose command line and memory stats. TODO: move over to gostrich so those can be viz-ed.
import _ "expvar"
import _ "net/http/pprof"

var (
	statsSingletonLock = sync.Mutex{}
	statsSingleton     *statsRecord        // singleton stats, that's "typically" what you need
	startUpTime        = time.Now().Unix() // start up time

	// command line arguments that can be used to customize this module's singleton
	adminPort       = flag.String("admin_port", "8300", "admin port")
	jsonLineBreak   = flag.Bool("json_line_break", true, "whether break lines for json")
	statsSampleSize = flag.Int("stats_sample_size", 1001, "how many samples to keep for stats")
)

type Admin interface {
	StartToLive() error
	GetStats() Stats
}

/*
 * Counter represents a thread safe way of keeping track of a single count.
 */
type Counter interface {
	Incr(by int64) int64
	Get() int64
}

/*
 * Sampler maintains a sample of input stream of numbers.
 */
type IntSampler interface {
	Observe(f int)
	Sampled() []int
}

/*
 * One implementation of sampler, it does so by keeping track of last n items. It also keeps
 * track of overall count and sum, so historical average can be calculated.
 */
type intSampler struct {
	// historical
	count  int64
	sum    int64
	length int

	// thread safe buffer
	cache []int
}

type intSamplerWithClone struct {
	intSampler
	// cloned cache's used to do stats reporting, where we need to sort the content of cache.
	clonedCache []int
}

/*
 * The interface used to collect various stats. It provides counters, gauges, labels and samples.
 * It also provides a way to scope Stats collector to a prefixed domain. All implementation should
 * be thread safe.
 */
type Stats interface {
	Counter(name string) Counter
	AddGauge(name string, gauge func() float64) bool
	AddLabel(name string, label func() string) bool
	Statistics(name string) IntSampler
	Scoped(name string) Stats
}

/*
 * myInt64 will be a Counter.
 */
type myInt64 int64

/*
 * will be a Stats
 */
type statsRecord struct {
	// Global lock's bad, user should keep references to actual collectors, such as Counters
	// instead of doing name resolution each every time.
	lock sync.RWMutex

	counters map[string]*int64
	gauges   map[string]func() float64
	labels   map[string]func() string

	samplerSize int // val
	statistics  map[string]*intSamplerWithClone
}

/*
 * statsRecord with a scope name, it prefix all stats with this scope name.
 */
type scopedStatsRecord struct {
	base  *statsRecord
	scope string
}

/*
 * Stats that can be served through HTTP
 */
type statsHttp struct {
	*statsRecord
	address string
}

/*
 * Serves Json endpoint.
 */
type statsHttpJson struct {
	*statsHttp
	jsonLineBreak bool
}

/*
 * Serves Txt endpoint.
 */
type statsHttpTxt statsHttp

/*
 * Creates a sampler of given size
 */
func NewIntSampler(size int) *intSampler {
	return &intSampler{
		0,
		0,
		size,
		make([]int, size),
	}
}

func NewIntSamplerWithClone(size int) *intSamplerWithClone {
	return &intSamplerWithClone{
		intSampler {
			0,
			0,
			size,
			make([]int, size),
		},
		make([]int, size),
	}
}

func (s *intSampler) Observe(f int) {
	count := atomic.AddInt64(&(s.count), 1)
	atomic.AddInt64(&(s.sum), int64(f))
	s.cache[int((count-1)%int64(s.length))] = f
}

func (s *intSampler) Sampled() []int {
	n := s.count
	if n < int64(s.length) {
		return s.cache[0:n]
	}
	return s.cache
}

func (s *intSamplerWithClone) Observe(f int) {
	count := atomic.AddInt64(&(s.count), 1)
	atomic.AddInt64(&(s.sum), int64(f))
	s.cache[int((count-1)%int64(s.length))] = f
}

func (s *intSamplerWithClone) Sampled() []int {
	n := s.count
	if n < int64(s.length) {
		return s.cache[0:n]
	}
	return s.cache
}

/*
 * Create a new stats object
 */
func NewStats(sampleSize int) *statsRecord {
	return &statsRecord{
		sync.RWMutex{},
		make(map[string]*int64),
		make(map[string]func() float64),
		make(map[string]func() string),
		sampleSize,
		make(map[string]*intSamplerWithClone),
	}
}

func (sr *statsRecord) Counter(name string) Counter {
	sr.lock.RLock()
	if v, ok := sr.counters[name]; ok {
		sr.lock.RUnlock()
		return (*myInt64)(v)
	}
	sr.lock.RUnlock()

	sr.lock.Lock()
	defer sr.lock.Unlock()

	if v, ok := sr.counters[name]; ok {
		return (*myInt64)(v)
	}

	var v int64
	vv := &v
	sr.counters[name] = vv
	return (*myInt64)(vv)
}

func (sr *statsRecord) AddGauge(name string, gauge func() float64) bool {
	sr.lock.RLock()
	if _, ok := sr.gauges[name]; ok {
		sr.lock.RUnlock()
		return false
	}
	sr.lock.RUnlock()

	sr.lock.Lock()
	defer sr.lock.Unlock()

	if _, ok := sr.gauges[name]; ok {
		return false
	}

	sr.gauges[name] = gauge
	return true
}

func (sr *statsRecord) AddLabel(name string, label func() string) bool {
	sr.lock.RLock()
	if _, ok := sr.labels[name]; ok {
		sr.lock.RUnlock()
		return false
	}
	sr.lock.RUnlock()

	sr.lock.Lock()
	defer sr.lock.Unlock()

	if _, ok := sr.labels[name]; ok {
		return false
	}

	sr.labels[name] = label
	return true
}

func (sr *statsRecord) Statistics(name string) IntSampler {
	sr.lock.RLock()
	if v, ok := sr.statistics[name]; ok {
		sr.lock.RUnlock()
		return (v)
	}
	sr.lock.RUnlock()

	sr.lock.Lock()
	defer sr.lock.Unlock()

	if v, ok := sr.statistics[name]; ok {
		return (v)
	}

	vv := NewIntSamplerWithClone(sr.samplerSize)
	sr.statistics[name] = vv
	return vv
}

func (sr *statsRecord) Scoped(name string) Stats {
	return &scopedStatsRecord{
		sr,
		name,
	}
}

func (ssr *scopedStatsRecord) Counter(name string) Counter {
	return ssr.base.Counter(ssr.scope + "/" + name)
}

func (ssr *scopedStatsRecord) AddGauge(name string, gauge func() float64) bool {
	return ssr.base.AddGauge(ssr.scope+"/"+name, gauge)
}

func (ssr *scopedStatsRecord) AddLabel(name string, label func() string) bool {
	return ssr.base.AddLabel(ssr.scope+"/"+name, label)
}
func (ssr *scopedStatsRecord) Statistics(name string) IntSampler {
	return ssr.base.Statistics(ssr.scope + "/" + name)
}

func (ssr *scopedStatsRecord) Scoped(name string) Stats {
	return &scopedStatsRecord{
		ssr.base,
		ssr.scope + "/" + name,
	}
}

func (c *myInt64) Incr(by int64) int64 {
	return atomic.AddInt64((*int64)(c), by)
}

func (c *myInt64) Get() int64 {
	return int64(*c)
}

type sortedValues struct {
	name   string
	values []float64
}

/*
 * Output a sorted array of float64 as percentile in Json format.
 */
func sortedToJson(w http.ResponseWriter, array []int, count int64, sum int64) {
	fmt.Fprintf(w, "{")
	length := len(array)
	l1 := length - 1
	if length > 0 {
		// historical
		fmt.Fprintf(w, "\"count\":%v,", count)
		fmt.Fprintf(w, "\"sum\":%v,", sum)
		fmt.Fprintf(w, "\"average\":%v,", float64(sum)/float64(count))

		// percentile
		fmt.Fprintf(w, "\"minimum\":%v,", array[0])
		fmt.Fprintf(w, "\"p25\":%v,", array[int(math.Min(0.25*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p50\":%v,", array[int(math.Min(0.50*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p75\":%v,", array[int(math.Min(0.75*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p90\":%v,", array[int(math.Min(0.90*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p99\":%v,", array[int(math.Min(0.99*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p999\":%v,", array[int(math.Min(0.999*float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"maximum\":%v", array[l1])
	}
	fmt.Fprintf(w, "}")
}

/*
 * Output a sorted array of float64 as percentile in text format.
 */
func sortedToTxt(w http.ResponseWriter, array []int, count int64, sum int64) {
	length := len(array)
	l1 := length - 1
	fmt.Fprintf(w, "(")
	if length > 0 {
		// historical
		fmt.Fprintf(w, "count=%v, ", count)
		fmt.Fprintf(w, "sum=%v, ", sum)
		fmt.Fprintf(w, "average=%v, ", float64(sum)/float64(count))

		// percentile
		fmt.Fprintf(w, "minimum=%v, ", array[0])
		fmt.Fprintf(w, "p25=%v, ", array[int(math.Min(0.25*float64(length), float64(l1)))])
		fmt.Fprintf(w, "p50=%v, ", array[int(math.Min(0.50*float64(length), float64(l1)))])
		fmt.Fprintf(w, "p75=%v, ", array[int(math.Min(0.75*float64(length), float64(l1)))])
		fmt.Fprintf(w, "p90=%v, ", array[int(math.Min(0.90*float64(length), float64(l1)))])
		fmt.Fprintf(w, "p99=%v, ", array[int(math.Min(0.99*float64(length), float64(l1)))])
		fmt.Fprintf(w, "p999=%v, ", array[int(math.Min(0.999*float64(length), float64(l1)))])

		fmt.Fprintf(w, "maximum=%v", array[l1])
	}
	fmt.Fprintf(w, ")")
}

func jsonEncode(v interface{}) string {
	if b, err := json.Marshal(v); err == nil {
		return string(b)
	}
	return "bad_json_value"
}

func (sr *statsHttpJson) breakLines() string {
	if sr.jsonLineBreak {
		return "\n"
	}
	return ""
}

/*
 * High perf freeze content of a sampler and sort it
 */
func freezeAndSort(s *intSamplerWithClone) (int64, int64, []int) {
	// freeze, there might be a drift, we are fine
	count := s.count
	sum := s.sum

	// copy cache
	for i, a := range s.cache {
		s.clonedCache[i] = a
	}
	v := s.clonedCache
	if count < int64(s.length) {
		v = s.cache[0:int(count)]
	}
	sort.Ints(v)
	return count, sum, v
}

func (sr *statsHttpJson) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{"+sr.breakLines())
	first := true
	// counters
	for k, v := range sr.counters {
		if !first {
			fmt.Fprintf(w, ","+sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), *v)
	}
	// gauges
	for k, f := range sr.gauges {
		if !first {
			fmt.Fprintf(w, ","+sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), f())
	}
	// labels
	for k, f := range sr.labels {
		if !first {
			fmt.Fprintf(w, ","+sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), jsonEncode(f()))
	}
	// stats
	for k, v := range sr.statistics {
		count, sum, vv := freezeAndSort(v)
		if count > 0 {
			if !first {
				fmt.Fprintf(w, ","+sr.breakLines())
			}
			first = false
			fmt.Fprintf(w, "%v: ", jsonEncode(k))
			sortedToJson(w, vv, count, sum)
			fmt.Fprintf(w, "\n")
		}
	}
	fmt.Fprintf(w, sr.breakLines()+"}"+sr.breakLines())
}

func (sr *statsHttpTxt) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	sr.lock.RLock()
	defer sr.lock.RUnlock()

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// counters
	for k, v := range sr.counters {
		fmt.Fprintf(w, "%v: %v\n", k, *v)
	}
	// gauges
	for k, f := range sr.gauges {
		fmt.Fprintf(w, "%v: %v\n", k, f())
	}
	// labels
	for k, f := range sr.labels {
		fmt.Fprintf(w, "%v: %v\n", k, f())
	}
	// stats
	for k, v := range sr.statistics {
		count, sum, vv := freezeAndSort(v)
		if count > 0 {
			fmt.Fprintf(w, "%v: ", k)
			sortedToTxt(w, vv, count, sum)
			fmt.Fprintf(w, "\n")
		}
	}
}

func init() {
}

type AdminError string

func (e AdminError) Error() string {
	return string(e)
}

func (stats *statsRecord) GetStats() Stats {
	return stats
}

/*
 * Blocks current coroutine. Call http /shutdown to shutdown.
 */
func (stats *statsRecord) StartToLive(adminPort *string, jsonLineBreak *bool) error {
	// only start a single copy
	statsHttpImpl := &statsHttp{stats, ":" + *adminPort}
	statsJson := &statsHttpJson{statsHttpImpl, *jsonLineBreak}
	statsTxt := (*statsHttpTxt)(statsHttpImpl)

	shutdown := make(chan int)
	serverError := make(chan error)

	mux := http.NewServeMux()
	mux.Handle("/stats.json", statsJson)
	mux.Handle("/stats.txt", statsTxt)
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request) {
		shutdown <- 0
	})

	server := http.Server{
		statsHttpImpl.address,
		mux,
		18 * time.Second,
		10 * time.Second,
		0,
		nil,
	}

	go func() {
		serverError <- server.ListenAndServe()
	}()

	select {
	case er := <-serverError:
		return AdminError("Can't start up server, error was: " + er.Error())
	case <-shutdown:
		log.Println("Shutdown requested")
	}
	return nil
}

func StatsSingleton() Stats {
	statsSingletonLock.Lock()
	defer statsSingletonLock.Unlock()
	if statsSingleton == nil {
		statsSingleton = NewStats(*statsSampleSize)

		// some basic stats
		statsSingleton.AddGauge("uptime", func() float64 {
			return float64(time.Now().Unix() - startUpTime)
		})
	}
	return statsSingleton
}

func StartToLive() error {
	// starts up debugging server
	go func() {
		log.Println(http.ListenAndServe(":6060", nil))
	}()
	//making sure stats are created.
	StatsSingleton()
	return statsSingleton.StartToLive(adminPort, jsonLineBreak)
}
