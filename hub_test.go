package conc

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"log"
	"sort"
	"testing"
	"time"
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
		return func(msg ValueOrError[Msg]) error {
			msgstr := fmt.Sprintf("%03d - %s - %s", msg.Value.value, msg.Value.topic, name)
			results = append(results, msgstr)
			// log.Printf("Received: %s", msgstr)
			return nil
		}
	}
	c1, _ := hub.Connect(nil, makewriter("c1"))
	c1.Subscribe("a", "b", "c", "d")
	c2, _ := hub.Connect(nil, makewriter("c2"))
	c2.Subscribe("c", "d", "e", "f")
	c3, _ := hub.Connect(nil, makewriter("c3"))
	c3.Subscribe("e", "f", "g", "h")

	callback := make(chan Msg)
	defer close(callback)
	hub.Send(ValueOrError[Msg]{Value: Msg{"a", 1}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"b", 2}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"c", 3}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"d", 4}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"e", 5}}, callback)
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
	hub.Send(ValueOrError[Msg]{Value: Msg{"a", 6}}, callback)
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
		return func(msg ValueOrError[Msg]) error {
			msgstr := fmt.Sprintf("%03d - %s - %s", msg.Value.value, msg.Value.topic, name)
			results = append(results, msgstr)
			// log.Printf("Received: %s", msgstr)
			return nil
		}
	}
	c1, _ := hub.Connect(nil, makewriter("c1"))
	c1.Subscribe("a", "b", "c", "d")
	c2, _ := hub.Connect(nil, makewriter("c2"))
	c2.Subscribe("c", "d", "e", "f")
	c3, _ := hub.Connect(nil, makewriter("c3"))
	c3.Subscribe("e", "f", "g", "h")

	callback := make(chan Msg)
	defer close(callback)
	hub.Send(ValueOrError[Msg]{Value: Msg{"a", 1}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"b", 2}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"c", 3}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"d", 4}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"e", 5}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"f", 6}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"g", 7}}, nil)
	hub.Send(ValueOrError[Msg]{Value: Msg{"h", 8}}, callback)
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
	hub.Send(ValueOrError[Msg]{Value: Msg{"a", 9}}, callback)
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

func TestHubWithReaders(t *testing.T) {
	log.Println("===================== TestHubWithReaders =====================")
	hub := NewHub[Msg](nil)

	var results []string
	makewriter := func(name string) HubWriter[Msg] {
		return func(msg ValueOrError[Msg]) error {
			msgstr := fmt.Sprintf("%03d - %s - %s", msg.Value.value, msg.Value.topic, name)
			results = append(results, msgstr)
			// log.Printf("Received: %s", msgstr)
			return nil
		}
	}
	makereader := func(name string, start, end int) HubReader[Msg] {
		curr := start
		return func() (msg ValueOrError[Msg]) {
			if curr <= end {
				curr += 1
			} else {
				return ValueOrError[Msg]{Value: Msg{topic: name, value: end + 1}, Closed: true}
			}
			return ValueOrError[Msg]{Value: Msg{
				topic: name,
				value: curr - 1,
			}}
		}
	}
	c1, _ := hub.Connect(nil, makewriter("c1"))
	c1.Subscribe("a", "b", "c", "d")
	c2, _ := hub.Connect(nil, makewriter("c2"))
	c2.Subscribe("c", "d", "e", "f")
	c3, _ := hub.Connect(nil, makewriter("c3"))
	c3.Subscribe("e", "f", "g", "h")

	hub.Connect(makereader("r1", 1, 5), nil)
	hub.Connect(makereader("r2", 10, 15), nil)
	// log.Println("Result after 8 -> h: ", results)

	log.Println("results: ", results)
	// hub.Stop()

	time.Sleep(time.Second * 1)
	sort.Strings(results)
	assert.Equal(t, results, []string{
		"001 - r1 - c1",
		"001 - r1 - c2",
		"001 - r1 - c3",
		"002 - r1 - c1",
		"002 - r1 - c2",
		"002 - r1 - c3",
		"003 - r1 - c1",
		"003 - r1 - c2",
		"003 - r1 - c3",
		"004 - r1 - c1",
		"004 - r1 - c2",
		"004 - r1 - c3",
		"005 - r1 - c1",
		"005 - r1 - c2",
		"005 - r1 - c3",
		"006 - r1 - c1",
		"006 - r1 - c2",
		"006 - r1 - c3",
		"010 - r2 - c1",
		"010 - r2 - c2",
		"010 - r2 - c3",
		"011 - r2 - c1",
		"011 - r2 - c2",
		"011 - r2 - c3",
		"012 - r2 - c1",
		"012 - r2 - c2",
		"012 - r2 - c3",
		"013 - r2 - c1",
		"013 - r2 - c2",
		"013 - r2 - c3",
		"014 - r2 - c1",
		"014 - r2 - c2",
		"014 - r2 - c3",
		"015 - r2 - c1",
		"015 - r2 - c2",
		"015 - r2 - c3",
		"016 - r2 - c1",
		"016 - r2 - c2",
		"016 - r2 - c3",
	})
}
