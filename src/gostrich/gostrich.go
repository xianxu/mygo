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
)

// expose command line and memory stats. TODO: move over to gostrich so those can be viz-ed.
import _ "expvar"

var (
	StatsSingleton *statsRecord         // singleton stats, that's "typically" what you need
	startUpTime    = time.Now().Unix()  // start up time

	// command line arguments that can be used to customize this module's singleton
	adminPort      = flag.String("admin_port", "8300", "admin port")
	jsonLineBreak  = flag.Bool("json_line_break", true, "whether break lines for json")
)

type Admin interface {
	StartToLive() error
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
type Sampler interface {
	Observe(f float64)
	Sampled() []float64
}

/*
 * One implementation of sampler, it does so by keeping track of min/max and last n items.
 */
type sampler struct {
	count  int64
	cache  []float64
	min    float64
	max    float64
	length int
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
	Statistics(name string) Sampler
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
	lock        sync.Mutex

	counters    map[string]*int64
	gauges      map[string]func() float64
	labels      map[string]func() string

	samplerSize int
	statistics  map[string]*sampler

	// stats are periodically cloned here so that they can be sorted and such without locking.
	// TODO: finish
	clonedStats map[string]*sampler
}

/*
 * statsRecord with a scope name, it prefix all stats with this scope name.
 */
type scopedStatsRecord struct {
	base        *statsRecord
	scope       string
}

/*
 * Stats that can be served through HTTP
 */
type statsHttp struct {
	*statsRecord
	address     string
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
type statsHttpTxt  statsHttp

/*
 * Creates a sampler of given size
 */
func NewSampler(size int) *sampler {
	return &sampler{0, make([]float64, size), math.MaxFloat64, -1 * math.MaxFloat64, size}
}

func (s *sampler) Observe(f float64) {
	count := atomic.AddInt64(&(s.count), 1)
	s.cache[int((count - 1) % int64(s.length))] = f
}

func (s *sampler) Sampled() []float64 {
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
		sync.Mutex{},
		make(map[string]*int64),
		make(map[string]func() float64),
		make(map[string]func() string),
		sampleSize,
		make(map[string]*sampler),
		make(map[string]*sampler),
	}
}

func (sr *statsRecord) Counter(name string) Counter {
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
	sr.lock.Lock()
	defer sr.lock.Unlock()

	if _, ok := sr.gauges[name]; ok {
		return false
	}

	sr.gauges[name] = gauge
	return true
}

func (sr *statsRecord) AddLabel(name string, label func() string) bool {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	if _, ok := sr.labels[name]; ok {
		return false
	}

	sr.labels[name] = label
	return true
}

func (sr *statsRecord) Statistics(name string) Sampler {
	fmt.Printf("There are %v stats before adding %v\n", len(sr.statistics), name)
	sr.lock.Lock()
	defer sr.lock.Unlock()

	if v, ok := sr.statistics[name]; ok {
		return (v)
	}

	vv := NewSampler(sr.samplerSize)
	sr.statistics[name] = vv
	return vv
}

func (sr *statsRecord) Scoped(name string) Stats {
	return &scopedStatsRecord {
		sr,
		name,
	}
}

func (ssr *scopedStatsRecord) Counter(name string) Counter {
	return ssr.base.Counter(ssr.scope + "/" + name)
}

func (ssr *scopedStatsRecord) AddGauge(name string, gauge func() float64) bool {
	return ssr.base.AddGauge(ssr.scope + "/" + name, gauge)
}

func (ssr *scopedStatsRecord) AddLabel(name string, label func() string) bool {
	return ssr.base.AddLabel(ssr.scope + "/" + name, label)
}
func (ssr *scopedStatsRecord) Statistics(name string) Sampler {
	return ssr.base.Statistics(ssr.scope + "/" + name)
}

func (ssr *scopedStatsRecord) Scoped(name string) Stats {
	return &scopedStatsRecord {
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
func sortedToJson(w http.ResponseWriter, array []float64) {
	fmt.Fprintf(w, "{")
	length := len(array)
	l1 := length - 1
	if length > 0 {
		fmt.Fprintf(w, "\"count\":%v,",length)
		sum := 0.0
		for _, v := range array {
			sum += v
		}
		fmt.Fprintf(w, "\"sum\":%v,",sum)
		fmt.Fprintf(w, "\"average\":%v,",sum/float64(length))
		fmt.Fprintf(w, "\"minimum\":%v,",array[0])
		fmt.Fprintf(w, "\"p25\":%v,",array[int(math.Min(0.25 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p50\":%v,",array[int(math.Min(0.50 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p75\":%v,",array[int(math.Min(0.75 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p90\":%v,",array[int(math.Min(0.90 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p99\":%v,",array[int(math.Min(0.99 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "\"p999\":%v,",array[int(math.Min(0.999 * float64(length), float64(l1)))])

		fmt.Fprintf(w, "\"maximum\":%v",array[l1])
	}
	fmt.Fprintf(w, "}")
}

/*
 * Output a sorted array of float64 as percentile in text format.
 */
func sortedToTxt(w http.ResponseWriter, array []float64) {
	length := len(array)
	l1 := length - 1
	fmt.Fprintf(w, "(")
	if length > 0 {
		fmt.Fprintf(w, "count=%v, ",length)
		sum := 0.0
		for _, v := range array {
			sum += v
		}
		fmt.Fprintf(w, "sum=%v, ",sum)
		fmt.Fprintf(w, "average=%v, ",sum/float64(length))
		fmt.Fprintf(w, "minimum=%v, ",array[0])
		fmt.Fprintf(w, "p25=%v, ",array[int(math.Min(0.25 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "p50=%v, ",array[int(math.Min(0.50 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "p75=%v, ",array[int(math.Min(0.75 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "p90=%v, ",array[int(math.Min(0.90 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "p99=%v, ",array[int(math.Min(0.99 * float64(length), float64(l1)))])
		fmt.Fprintf(w, "p999=%v, ",array[int(math.Min(0.999 * float64(length), float64(l1)))])

		fmt.Fprintf(w, "maximum=%v",array[l1])
	}
	fmt.Fprintf(w, ")")
}

func (sr *statsRecord) cloneAll() {
	sr.lock.Lock()
	defer sr.lock.Unlock()

	//TODO:
	//  - actually copy, note there's no need to lock, if we can tolerate a bit inaccuracy in
	//    stats, such as max is not really max, but it's large enough typically.
	//  - maintain a single buffer of copied values
	//  - let's do single threaded operation for easier buffer management

	for k, v := range sr.statistics {
		if vv, ok := sr.clonedStats[k]; ok {
			vv.count = v.count
			vv.min = v.min
			vv.max = v.max
			vv.length = v.length
			for i, a := range v.cache {
				vv.cache[i] = a
			}
		} else {
			c := make([]float64, v.length)
			for i, a := range v.cache {
				c[i] = a
			}
			sr.clonedStats[k] = &sampler{
				v.count,
				c,
				v.min,
				v.max,
				v.length,
			}
		}
	}
}

func (sr *statsRecord) sortAll() {
	for _, v := range sr.clonedStats {
        toSort := v.cache
		if v.count < int64(v.length) {
			toSort = v.cache[0:int(v.count)]
		}
		sort.Float64s(toSort)
	}
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
func (sr *statsHttpJson) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("admin serving http")
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{" + sr.breakLines())
	first := true
	fmt.Println(1)
	// counters
	for k, v := range sr.counters {
		if !first {
			fmt.Fprintf(w, "," + sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), *v)
	}
	fmt.Println(2)
	// gauges
	for k, f := range sr.gauges {
		if !first {
			fmt.Fprintf(w, "," + sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), f())
	}
	fmt.Println(3)
	// labels
	for k, f := range sr.labels {
		if !first {
			fmt.Fprintf(w, "," + sr.breakLines())
		}
		first = false
		fmt.Fprintf(w, "%v: %v", jsonEncode(k), jsonEncode(f()))
	}
	fmt.Println(4)
	// stats
	sr.cloneAll()
	sr.sortAll()
	for k, v := range sr.clonedStats {
		if v.count > 0 {
			fmt.Println("sort collected")
			if !first {
				fmt.Fprintf(w, "," + sr.breakLines())
			}
			first = false
			fmt.Fprintf(w, "%v: ", jsonEncode(k))
			vv := v.cache
			if v.count < int64(v.length) {
				vv = v.cache[0:int(v.count)]
			}
			sortedToJson(w, vv)
			fmt.Fprintf(w, "\n")
		}
	}
	fmt.Fprintf(w, sr.breakLines() + "}" + sr.breakLines())
}

func (sr *statsHttpTxt) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
	sr.cloneAll()
	sr.sortAll()
	for k, v := range sr.clonedStats {
		fmt.Printf("sort collected: len %v\n", v.count)
		if v.count > 0 {
			fmt.Fprintf(w, "%v: ", k)
			vv := v.cache
			if v.count < int64(v.length) {
				vv = v.cache[0:int(v.count)]
			}
			sortedToTxt(w, vv)
			fmt.Fprintf(w, "\n")
		}
	}
}

func init() {
	StatsSingleton = NewStats(1001)

	// some basic stats
	StatsSingleton.AddGauge("uptime", func()float64 {
		return float64(time.Now().Unix() - startUpTime)
	})
}

type AdminError string
func (e AdminError) Error() string {
	return string(e)
}

/*
 * Blocks current coroutine. Call http /shutdown to shutdown.
 */
func (stats *statsRecord) StartToLive(adminPort *string, jsonLineBreak *bool) error {
	// only start a single copy
	statsHttpImpl := &statsHttp{ stats, ":" + *adminPort }
	statsJson := &statsHttpJson{ statsHttpImpl, *jsonLineBreak }
	statsTxt := (*statsHttpTxt)(statsHttpImpl)

	shutdown    := make(chan int)
	serverError := make(chan error)

	mux := http.NewServeMux()
	mux.Handle("/stats.json", statsJson)
	mux.Handle("/stats.txt", statsTxt)
	mux.HandleFunc("/shutdown", func(w http.ResponseWriter, r *http.Request){
		shutdown <- 0
	})

	server := http.Server {
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
		fmt.Println("Shutdown requested")
	}
	return nil
}

func StartToLive() error {
	return StatsSingleton.StartToLive(adminPort, jsonLineBreak)
}

/*
* Some random test code, not sure where to put them yet. TODO: real tests
 */
func test() {
	flag.Parse()
	stats := StatsSingleton

	g1 := float64(0)
	stats.Counter("c1").Incr(1)
	stats.Counter("c1").Incr(1)
	stats.AddGauge("g1", func() float64 {
		return g1
	})

	fmt.Printf("Yo there %d\n", stats.Counter("c1").Get())
	fmt.Printf("Yo there %d\n", stats.Counter("g1").Get())
	s := NewSampler(3)
	s.Observe(1)
	s.Observe(1)
	s.Observe(1)
	s.Observe(2)
	s.Observe(2)
	tflock := stats.Statistics("tflock")
	for i := 1; i < 2000; i += 1 {
		tflock.Observe(float64(i))
	}
	stats.AddGauge("yo", func() float64 { return float64(time.Now().Second()) })
	stats.AddLabel("hello", func() string { return "world" })
	fmt.Println(s.Sampled())

	ms := stats.Scoped("memcache_client")
	ms.Counter("requests").Incr(1)
	ms.Counter("requests").Incr(1)
	ms1 := ms.Scoped("client1")
	ms1.Counter("requests").Incr(1)

	stats.StartToLive(adminPort, jsonLineBreak)
}
