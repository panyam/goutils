package conc

import (
	"github.com/stretchr/testify/assert"
	"log"
	"sort"
	"sync"
	"testing"
)

func TestFanIn(t *testing.T) {
	log.Println("===================== TestFanIn =====================")
	inch := []chan int{
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan int),
	}
	outch := make(chan int)
	fanin := NewFanIn(outch)

	var vals []int
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		n := 0
		for fanin.IsRunning() {
			i := <-fanin.Channel()
			log.Println("Received: ", i)
			vals = append(vals, i)
			n += 1
			if n >= 15 {
				fanin.Stop()
			}
		}
		wg.Done()
	}()

	for ch := 0; ch < 5; ch++ {
		fanin.Add(inch[ch])
		for msg := 0; msg < 3; msg++ {
			v := ch*3 + msg
			log.Println("Writing values: ", v)
			inch[ch] <- v
		}
	}
	log.Println("Waiting to finish...")
	wg.Wait()

	// Sort since fanin can combine in any order
	sort.Ints(vals)
	for i := 0; i < 15; i++ {
		assert.Equal(t, vals[i], i, "Out vals dont match")
	}
}
