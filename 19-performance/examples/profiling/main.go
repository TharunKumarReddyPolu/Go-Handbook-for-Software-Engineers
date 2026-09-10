// Command profiling is a deliberately slow service for practicing the
// pprof workflow in 19-performance/01-measure-first.md. Every slow path
// is annotated with what you should see in profiles.
package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	_ "net/http/pprof" // registers /debug/pprof/* on the default mux
	"strconv"
	"sync"
)

// cpuHog burns cycles with pointless math — shows up as flat time in
// `go tool pprof top`.
func cpuHog(n int) int {
	total := 0
	for i := 0; i < n; i++ {
		total += i % 7
	}
	return total
}

// allocHog churns allocations — shows up in alloc_space/alloc_objects
// heap profiles and as GC work.
var allocHogResults sync.Map // pretend-cache so live set is observable

func allocHog(n int) []string {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, "item-"+strconv.Itoa(i))
	}
	return out
}

func main() {
	mux := http.NewServeMux()

	mux.HandleFunc("/work", func(w http.ResponseWriter, r *http.Request) {
		n, _ := strconv.Atoi(r.URL.Query().Get("n"))
		if n == 0 {
			n = 100000
		}
		_ = cpuHog(n)                                     // CPU profile: flat in cpuHog
		allocHogResults.Store(rand.Int63(), allocHog(64)) // heap: alloc churn + live growth
		fmt.Fprintln(w, "done")
	})

	// pprof endpoints on a separate port; protect in real deployments.
	go func() {
		log.Println("pprof on http://localhost:6060/debug/pprof/")
		_ = http.ListenAndServe("localhost:6060", nil)
	}()

	log.Println("work on http://localhost:8080/work?n=100000")
	log.Fatal(http.ListenAndServe(":8080", mux))
}
