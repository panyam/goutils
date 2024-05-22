/*
This package contains a few utilities to make it easier to make and send http requests and
handle the http responses.  Another important part of this package is simple to use wrapper over Gorilla websockets so that both the client and server loops can be written in a uniform way.

# Utilities to wrap websocket server and clients in a uniform way.

Gorilla websockets is an amazing package to bring websocket functionality to your application.  It is rather barebones (but extremely robust).
We want some extra properties from our Websockets like:

 1. Typed messages

 2. Customized pings/pongs with tuneable timeouts

Let us look at an example (available at cmd/timews/main.go).   We want to build a very simple websocket endpoint that sends out the current time periodically seconds to connected subscribers.   The subscribers can also publish a message that will be broadcast to all other connected subscribers (via a simple GET request).   We would need two endpoints for this:

  - /subscribe:
    This endpoint lets a client connect to the websocket endpoint and subscribe to messages.

  - /publish:
    The publish endpoint is used by clients to broadcast an arbitrary message to all connected clients.

Let us start with the main function and setup these routes.  This example uses the gorilla mux router to obtain request variables but any library accepting http.Handler should do.

	package main
	import (
		"fmt"
		"log"
		"net/http"
		"github.com/gorilla/mux"
	)

	func main() {
		r := mux.NewRouter()

		// Publish Handler
		r.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Publishing Custom Message")
		})

		// Subscribe Handler
		r.HandleFunc("/subscribe", func (w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Subscribing to get time")
		})

		srv := http.Server{Handler: r}
		log.Fatal(srv.ListenAndServe())
	}

Here the subscription is a normal http handler that returns a response.  However, we want this to be a websocket subscription handler.   So we need a couple helpers here:

1. A WSConn type to handle the lifecycle of this connection:

	// Our handler is the place to put all our "state" for this connection type
	type TimeHandler struct {
		// empty for now
	}

	// The Validate method gates the subscribe request to see if it should be upgraded
	// and if so creates the right connection type to wrap the connection
	// This examples allows all upgrades and is only needed to specify the kind of
	// connection type to use - in this case TimeConn.
	func (t *TimeHandler) Validate(w http.ResponseWriter, r *http.Request) (out *TimeConn, isValid bool) {
		return &TimeConn{handler: t}, true
	}

	// Our TimeConn allows us to override any connection instance specific behaviours
	type TimeConn struct {
	  ws.JSONConn
	  handler *TimeHandler
	}

We can now change our subscription http handler to:

	timeHandler := NewTimeHandler()
	r.HandleFunc("/subscribe", ws.WSServe(timeHandler, nil))

Now that we have a basic structure, we will use a conc.FanOut type to keep track of the list
of subscribers.  Update the TimeHandler to:

	type TimeHandler struct {
		Fanout *conc.FanOut[conc.Message[any]]
	}

	// ... along with a corresponding New method
	func NewTimeHandler() *TimeHandler {
		return &TimeHandler{Fanout: conc.NewFanOut[conc.Message[any]](nil)}
	}

The TimeHandler ensures that (in its Validate method) a new TimeConn is created to manage
the connection lifecycle.  We will now register the TimeConn's "output" channel into the
FanOut:

	func (t *TimeConn) OnStart(conn *websocket.Conn) error {
		t.JSONConn.OnStart(conn)
		writer := t.JSONConn.Writer

		log.Println("Got a new connection.....")
		// Register the writer channel into the fanout
		t.handler.Fanout.Add(writer.SendChan(), nil, false)
		return nil
	}

Similarly when a connection closes we want to de-register its output channel from the fanout:

	func (t *TimeConn) OnClose() {
		writer := t.JSONConn.Writer

		// Removal can be synchronous or asynchronous - we want to ensure it is done
		// synchronously so another publish (if one came in) wont be attempted on a closed channel
		<- t.handler.Fanout.Remove(writer.SendChan(), true)
		t.JSONConn.OnClose()
	}

Optional but we will disable timeouts from disconnecting our connection as we do not want to implement any client side logic (yet):

	func (t *TimeConn) OnTimeout() bool {
		return false
	}

That's all there is.   Create a websocket connection to ws://localhost/subscribe.   Easiest way is to use the tool websocat (https://github.com/vi/websocat) and:

	websocat ws://localhost/subscribe

You will note that nothing is printed.  That is because nothing is being published.  Let us update our main method to send messages on the Fanout:

	func main() {
		r := mux.NewRouter()
		timeHandler := NewTimeHandler()
		r.HandleFunc("/publish", func(w http.ResponseWriter, r *http.Request) {
			msg := r.URL.Query().Get("msg")
			timeHandler.Fanout.Send(conc.Message[any]{Value: fmt.Sprintf("%s: %s", time.Now().String(), msg)})
			fmt.Fprintf(w, "Published Message Successfully")
		})

		// Send the time every 1 second
		go func() {
			t := time.NewTicker(1 * time.Second)
			defer t.Stop()
			for {
				<-t.C
				timeHandler.Fanout.Send(conc.Message[any]{Value: time.Now().String()})
			}
		}()

		r.HandleFunc("/subscribe", gohttp.WSServe(timeHandler, nil))
		srv := http.Server{Handler: r}
		log.Fatal(srv.ListenAndServe())
	}

We have also updated the /publish handler to send custom messages on the fanout.

Now our subscriptions will show the time as well as custom publishes (curl http://localhost/publish?msg=YOUR_CUSTOM_MESSAGE)
*/
package http
