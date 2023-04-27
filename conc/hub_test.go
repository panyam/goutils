package conc

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"sort"
	/*
		"sync"
	*/
	"testing"
)

func TestHub(t *testing.T) {
	log.Println("===================== TestHub =====================")
	hub := NewHub[int](nil)

	var results []string
	makewriter := func(name string) HubWriter[int] {
		return func(msg HubMessage[int]) error {
			msgstr := fmt.Sprintf("%03d - %s", msg.Message, name)
			results = append(results, msgstr)
			log.Printf("Received: %s", msgstr)
			if msg.Callback != nil {
				msg.Callback <- msg.Message
			}
			return nil
		}
	}
	c1 := hub.Connect(makewriter("c1"))
	c1.Subscribe("a", "b", "c", "d")
	c2 := hub.Connect(makewriter("c2"))
	c2.Subscribe("c", "d", "e", "f")
	c3 := hub.Connect(makewriter("c3"))
	c3.Subscribe("e", "f", "g", "h")

	callback := make(chan int)
	defer close(callback)
	hub.Send("a", 1, nil)
	hub.Send("b", 2, nil)
	hub.Send("c", 3, nil)
	hub.Send("d", 4, nil)
	hub.Send("e", 5, nil)
	hub.Send("f", 6, nil)
	hub.Send("g", 7, nil)
	hub.Send("h", 8, callback)

	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - c1",
		"002 - c1",
		"003 - c1",
		"003 - c2",
		"004 - c1",
		"004 - c2",
		"005 - c2",
		"005 - c3",
		"006 - c2",
		"006 - c3",
		"007 - c3",
		"008 - c3",
	})
	<-callback
	log.Println("Result after 8 -> h: ", results)

	// Now try to remove subscriptions
	c1.Unsubscribe("a", "c")
	c2.Subscribe("a")
	hub.Send("a", 9, callback)
	<-callback
	log.Println("Result after 9 -> a: ", results)
	assert.Equal(t, results, []string{
		"001 - c1",
		"002 - c1",
		"003 - c1",
		"003 - c2",
		"004 - c1",
		"004 - c2",
		"005 - c2",
		"005 - c3",
		"006 - c2",
		"006 - c3",
		"007 - c3",
		"008 - c3",
		"009 - c2",
	})

	hub.Stop()
}
