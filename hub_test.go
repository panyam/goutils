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
		return func(msg int) error {
			msgstr := fmt.Sprintf("%03d - %s", msg, name)
			results = append(results, msgstr)
			log.Printf("Received: %s", msgstr)
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
	<-callback

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
	// log.Println("Result after 8 -> h: ", results)

	// Now try to remove subscriptions
	c1.Unsubscribe("a", "c")
	c2.Subscribe("a")
	hub.Send("a", 9, callback)
	<-callback
	// log.Println("Result after 9 -> a: ", results)
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

func TestHubWithBroadcaster(t *testing.T) {
	log.Println("===================== TestHubWithBroadcaster =====================")
	hub := NewHub[int](NewBroadcaster[int]())

	var results []string
	makewriter := func(name string) HubWriter[int] {
		return func(msg int) error {
			msgstr := fmt.Sprintf("%03d - %s", msg, name)
			results = append(results, msgstr)
			log.Printf("Received: %s", msgstr)
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
	hub.Send("d", 4, callback)
	<-callback

	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - c1",
		"001 - c2",
		"001 - c3",
		"002 - c1",
		"002 - c2",
		"002 - c3",
		"003 - c1",
		"003 - c2",
		"003 - c3",
		"004 - c1",
		"004 - c2",
		"004 - c3",
	})
	// log.Println("Result after 8 -> h: ", results)

	// Now try to remove subscriptions
	c1.Unsubscribe("a", "c")
	c2.Subscribe("a")
	hub.Send("a", 9, callback)
	<-callback
	// log.Println("Result after 9 -> a: ", results)
	assert.Equal(t, results, []string{
		"001 - c1",
		"001 - c2",
		"001 - c3",
		"002 - c1",
		"002 - c2",
		"002 - c3",
		"003 - c1",
		"003 - c2",
		"003 - c3",
		"004 - c1",
		"004 - c2",
		"004 - c3",
		"009 - c1",
		"009 - c2",
		"009 - c3",
	})

	hub.Stop()
}
