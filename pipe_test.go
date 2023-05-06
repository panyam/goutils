package conc

import (
	"github.com/stretchr/testify/assert"
	"log"
	"sync"
	"testing"
)

func TestPipe(t *testing.T) {
	log.Println("============== TestPipe ================")
	inch := make(chan int)
	outch := make(chan int)
	pipe := NewPipe(inch, outch, func(x int) int { return x * 2 })
	maxNums := 6
	var vals []int
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		for i := 0; i < maxNums; i++ {
			// log.Println("Written: ", i)
			inch <- i
		}
		pipe.Stop()
		wg.Done()
	}()

	for i := 0; i < maxNums; i++ {
		v := <-outch
		// log.Println("Read: ", v)
		vals = append(vals, v)
	}
	wg.Wait()

	for i := 0; i < maxNums; i++ {
		assert.Equal(t, vals[i], i*2, "Out vals dont match")
	}
}
