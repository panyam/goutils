package conc

import (
	"github.com/stretchr/testify/assert"
	"log"
	"testing"
)

func TestReader(t *testing.T) {
	log.Println("============== TestReader ================")
	inch := make(chan int)
	res := make(chan int, 5)
	makereader := func(ch chan int) *Reader[int] {
		return NewReader(func() (int, error) {
			val := <-ch
			return val, nil
		})
	}
	reader := makereader(inch)
	defer reader.Stop()

	var results []bool
	for i := 0; i < 5; i++ {
		results = append(results, false)
	}
	go func() {
		for {
			msg, _ := <-reader.RecvChan()
			results[msg.Value] = true
			res <- msg.Value
		}
	}()

	for i := 0; i < 5; i++ {
		inch <- i
	}

	for i := 0; i < 5; i++ {
		<-res
		assert.Equal(t, true, results[i], "Out vals dont match")
	}
}
