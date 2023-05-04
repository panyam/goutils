package conc

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"sort"
	"testing"
)

type Msg struct {
	topic string
	value int
}

func TestHubWithBroadcaster(t *testing.T) {
	log.Println("===================== TestHubWithBroadcaster =====================")
	hub := NewHub[Msg](nil)

	var results []string
	makewriter := func(name string) HubWriter[Msg] {
		return func(msg Msg, err error) error {
			msgstr := fmt.Sprintf("%03d - %s - %s", msg.value, msg.topic, name)
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

	callback := make(chan Msg)
	defer close(callback)
	hub.Send(Msg{"a", 1}, nil, nil)
	hub.Send(Msg{"b", 2}, nil, nil)
	hub.Send(Msg{"c", 3}, nil, nil)
	hub.Send(Msg{"d", 4}, nil, nil)
	hub.Send(Msg{"e", 5}, nil, callback)
	<-callback

	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - a - c1",
		"001 - a - c2",
		"001 - a - c3",
		"002 - b - c1",
		"002 - b - c2",
		"002 - b - c3",
		"003 - c - c1",
		"003 - c - c2",
		"003 - c - c3",
		"004 - d - c1",
		"004 - d - c2",
		"004 - d - c3",
		"005 - e - c1",
		"005 - e - c2",
		"005 - e - c3",
	})
	// log.Println("Result after 8 -> h: ", results)

	// Now try to remove subscriptions
	c1.Unsubscribe("a", "c")
	c2.Subscribe("a")
	hub.Send(Msg{"a", 6}, nil, callback)
	<-callback
	// log.Println("Result after 9 -> a: ", results)
	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - a - c1",
		"001 - a - c2",
		"001 - a - c3",
		"002 - b - c1",
		"002 - b - c2",
		"002 - b - c3",
		"003 - c - c1",
		"003 - c - c2",
		"003 - c - c3",
		"004 - d - c1",
		"004 - d - c2",
		"004 - d - c3",
		"005 - e - c1",
		"005 - e - c2",
		"005 - e - c3",
		"006 - a - c1",
		"006 - a - c2",
		"006 - a - c3",
	})

	hub.Stop()
}

func TestHubWithKVRouter(t *testing.T) {
	log.Println("===================== TestHubWithKVRouter =====================")
	router := NewKVRouter(func(msg Msg) TopicIdType {
		return msg.topic
	})
	hub := NewHub[Msg](router)

	var results []string
	makewriter := func(name string) HubWriter[Msg] {
		return func(msg Msg, err error) error {
			msgstr := fmt.Sprintf("%03d - %s - %s", msg.value, msg.topic, name)
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

	callback := make(chan Msg)
	defer close(callback)
	hub.Send(Msg{"a", 1}, nil, nil)
	hub.Send(Msg{"b", 2}, nil, nil)
	hub.Send(Msg{"c", 3}, nil, nil)
	hub.Send(Msg{"d", 4}, nil, nil)
	hub.Send(Msg{"e", 5}, nil, nil)
	hub.Send(Msg{"f", 6}, nil, nil)
	hub.Send(Msg{"g", 7}, nil, nil)
	hub.Send(Msg{"h", 8}, nil, callback)
	<-callback

	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - a - c1",
		"002 - b - c1",
		"003 - c - c1",
		"003 - c - c2",
		"004 - d - c1",
		"004 - d - c2",
		"005 - e - c2",
		"005 - e - c3",
		"006 - f - c2",
		"006 - f - c3",
		"007 - g - c3",
		"008 - h - c3",
	})
	// log.Println("Result after 8 -> h: ", results)

	// Now try to remove subscriptions
	c1.Unsubscribe("a", "c")
	c2.Subscribe("a")
	hub.Send(Msg{"a", 9}, nil, callback)
	<-callback
	// log.Println("Result after 9 -> a: ", results)
	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - a - c1",
		"002 - b - c1",
		"003 - c - c1",
		"003 - c - c2",
		"004 - d - c1",
		"004 - d - c2",
		"005 - e - c2",
		"005 - e - c3",
		"006 - f - c2",
		"006 - f - c3",
		"007 - g - c3",
		"008 - h - c3",
		"009 - a - c2",
	})

	hub.Stop()
}
