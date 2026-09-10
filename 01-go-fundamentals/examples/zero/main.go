// Command zero prints the zero value of the core types. Companion to
// 01-go-fundamentals/05-zero-values.md.
package main

import (
	"fmt"
	"time"
)

type Config struct {
	Retries int
	Timeout time.Duration
	TLS     bool
	Host    string
}

func main() {
	var (
		i    int
		f    float64
		b    bool
		s    string
		p    *int
		m    map[string]int
		sl   []int
		ch   chan int
		fn   func()
		anyv any
		c    Config
	)
	fmt.Printf("int:        %d\n", i)
	fmt.Printf("float64:    %g\n", f)
	fmt.Printf("bool:       %t\n", b)
	fmt.Printf("string:     %q (len %d)\n", s, len(s))
	fmt.Printf("*int:       %v\n", p)
	fmt.Printf("map:        %v (len %d, safe to read)\n", m, len(m))
	fmt.Printf("slice:      %v (len %d, safe to range)\n", sl, len(sl))
	fmt.Printf("chan:       %v (all ops block)\n", ch)
	fmt.Printf("func:       nil=%t\n", fn == nil)
	fmt.Printf("any:        %v\n", anyv)
	fmt.Printf("Config:     %+v\n", c)
}
