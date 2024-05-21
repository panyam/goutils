/*
This package contains a few utilities to make it easier to make and send http requests and
handle the http responses.

More importantly this package provides a simple to use wrapper over Gorilla websockets
so that both the client and server loops can be written in a uniform way.

Suppose we want to create the following message types in a websocket connection:

	type Message struct {
			Id string
			Content string
			Sender string
			CreatedAt uint64
			UpdatedAt uint64
	}

Gorilla websockets is an amazing package to bring websocket functionality to your application.  It is rather barebones (but extremely robust).
We want some extra properties from our Websockets like:

1. Typed messages
2. Customized pings/pongs with tuneable timeouts

Our websocket wrappers provide this and more.

# Associate a wbsocket h

First we want
First
*/

package http
