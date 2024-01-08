package conc

import (
	"log"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFanOut(t *testing.T) {
	fanout := NewFanOut[int](nil)
	var vals []int
	var wg sync.WaitGroup
	var m sync.Mutex
	for o := 0; o < 5; o++ {
		wg.Add(1)
		outch := fanout.New(nil)
		go func(o int, outch chan int) {
			defer fanout.Remove(outch, false)
			defer wg.Done()
			for count := 0; count < 10; count++ { //i fanout.IsRunning() {
				// log.Printf("Waiting to receive, in outch: %d", o)
				i := <-outch
				m.Lock()
				// log.Printf("Received %d, in outch: %d, Len: %d", i, o, len(vals))
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
	// log.Println("Vals: ", vals)
	for i := 0; i < 50; i++ {
		assert.Equal(t, vals[i], i/5, "Out vals dont match")
	}
}

func TestFanOut_WithClose(t *testing.T) {
	log.Println("===================== TestFanOutWithClose =====================")
	fanout := NewFanOut[int](nil)
	// A test where we simulate a hub like scenario where customers keep coming in and out (being added to and removed from the fanout)

	var writers []*Writer[int]

	var events []map[string]any
	var evLock sync.RWMutex
	addEvent := func(item map[string]any) {
		evLock.Lock()
		defer evLock.Unlock()
		item["time"] = time.Now()
		events = append(events, item)
	}

	SEND_INTERVAL := 10 * time.Millisecond
	WRITER_UPDATE_INTERVAL := 1 * time.Millisecond
	NUM_SENDS := 1000
	// One go routine that just sends messages to writers
	finished := false
	go func() {
		for i := 0; i < NUM_SENDS; i++ {
			addEvent(map[string]any{
				"fanout": fanout.DebugInfo(),
				"i":      i,
			})
			fanout.Send(i)
			time.Sleep(SEND_INTERVAL)
		}
		finished = true
	}()

	createWriter := func(writerId int) (out *Writer[int]) {
		out = NewWriter[int](func(value int) error {
			log.Println("Writer, Value, Ptr: ", writerId, value, out.SendChan())
			return nil
		})
		return
	}

	// controller to handle what fanout does etc
	writerId := 0
	for !finished {
		numWriters := len(writers)
		time.Sleep(WRITER_UPDATE_INTERVAL)
		add := numWriters < 5 || rand.Int()%2 == 0
		if add {
			writerId += 1
			writer := createWriter(writerId)
			writers = append(writers, writer)
			fanout.Add(writer.SendChan(), nil, false)
		} else {
			n := rand.Intn(numWriters)
			removedWriter := writers[n]
			writers[n] = writers[numWriters-1]
			writers = writers[:numWriters-1]
			// Remove first
			doneChan := fanout.Remove(removedWriter.SendChan(), true)
			<-doneChan
			// And *then* stop the writer
			removedWriter.Stop()
		}
	}
}
