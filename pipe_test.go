package conc

import (
	"log"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPipe(t *testing.T) {
	log.Println("============== TestPipe ================")
	inch := make(chan int)
	outch := make(chan int)
	pipe := NewPipe(inch, outch)
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
		close(inch)
		wg.Done()
	}()

	for i := 0; i < maxNums; i++ {
		v := <-outch
		// log.Println("Read: ", v)
		vals = append(vals, v)
	}
	wg.Wait()

	for i := 0; i < maxNums; i++ {
		assert.Equal(t, vals[i], i, "Out vals dont match")
	}
}

func TestReader2Pipe(t *testing.T) {
	log.Println("============== TestReader2Pipe ================")
	inch := make(chan int)
	outch := make(chan Message[int])
	makereader := func(ch chan int) *Reader[int] {
		return NewReader(func() (int, error, bool) {
			val := <-ch
			return val, nil, false
		})
	}
	reader := makereader(inch)
	pipe := NewPipe(reader.RecvChan(), outch)
	defer pipe.Stop()

	var results []bool
	for i := 0; i < 5; i++ {
		results = append(results, false)
	}
	go func() {
		for {
			msg, _ := <-outch
			results[msg.Value] = true
		}
	}()

	for i := 0; i < 5; i++ {
		inch <- i
	}
}
