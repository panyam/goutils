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
			i := <-fanin.RecvChan()
			// log.Println("Received: ", i)
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
			// log.Println("Writing values: ", v)
			inch[ch] <- v
		}
	}
	// log.Println("Waiting to finish...")
	wg.Wait()

	// Sort since fanin can combine in any order
	sort.Ints(vals)
	for i := 0; i < 15; i++ {
		assert.Equal(t, vals[i], i, "Out vals dont match")
	}
}

func TestMultiReadFanInToFanOut(t *testing.T) {
	log.Println("===================== TestMultiReadFanInToFanOut =====================")
	inch := []chan int{
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan int),
		make(chan int),
	}
	outch := make(chan int)
	fanin := NewFanIn(outch)

	for ch := 0; ch < 5; ch++ {
		fanin.Add(inch[ch])
	}

	var results []int
	var m sync.Mutex
	resch := make(chan int)
	writer := NewWriter(func(val int) error {
		log.Println("Here....", val)
		m.Lock()
		results = append(results, val)
		m.Unlock()
		resch <- val
		return nil
	})
	fanout := NewIDFanOut[int](nil, nil)
	fanout.Add(writer.SendChan(), nil)

	go func() {
		for {
			select {
			case val := <-fanin.RecvChan():
				fanout.Send(val)
				break
			}
		}
	}()

	// log.Println("Sending 5 values")
	for i := 0; i < 5; i++ {
		inch[i] <- i
	}
	// log.Println("Waiting 5 values")
	for i := 0; i < 5; i++ {
		<-resch
	}

	// log.Println("Results: ", results)
	assert.Equal(t, len(results), 5)
	fanin.Stop()
	fanout.Stop()
}
