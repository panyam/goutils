package conc

import (
	"github.com/stretchr/testify/assert"
	"log"
	"sort"
	"sync"
	"testing"
)

func TestFanOut(t *testing.T) {
	fanout := NewFanOut[int, int](nil)
	fanout.Mapper = IDFunc[int]
	var vals []int
	var wg sync.WaitGroup
	var m sync.Mutex
	for o := 0; o < 5; o++ {
		wg.Add(1)
		outch := fanout.New()
		go func(o int, outch <-chan int) {
			defer fanout.Remove(outch)
			defer wg.Done()
			for count := 0; count < 10; count++ { //i fanout.IsRunning() {
				// log.Printf("Waiting to receive, in outch: %d", o)
				i := <-outch
				m.Lock()
				log.Printf("Received %d, in outch: %d, Len: %d", i, o, len(vals))
				vals = append(vals, i)
				m.Unlock()
			}
		}(o, outch)
	}

	for i := 0; i < 10; i++ {
		fanout.Send(i)
	}
	wg.Wait()

	// Sort since fanout can combine in any order
	sort.Ints(vals)
	log.Println("Vals: ", vals)
	for i := 0; i < 50; i++ {
		assert.Equal(t, vals[i], i/5, "Out vals dont match")
	}
}
