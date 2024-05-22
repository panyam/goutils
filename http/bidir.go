package http

import "time"

// Configuration for a bidirectional "stream"
type BiDirStreamConfig struct {
	// Our connection can send pings at repeated intervals as a form of
	// healthcheck.  This property is a way to specify that duration.
	PingPeriod time.Duration

	// If no data (ping or otherwise) has been received from the remote
	// side within this duration then it is an indication for the handler to treat
	// this as a closed/timedout connection.  The handler an chose to terminate the connection
	// at this point by handling the OnTimeout method on the Conn interface.
	PongPeriod time.Duration
}

// Creates a bidirectional stream config with default values for ping and pong durations.
func DefaultBiDirStreamConfig() *BiDirStreamConfig {
	return &BiDirStreamConfig{
		PingPeriod: time.Second * 30,
		PongPeriod: time.Second * 300,
	}
}

type BiDirStreamConn[I any] interface {
	// Called to send the next ping message.
	SendPing() error

	// Optional Name of the connection
	Name() string

	// Optional connection id
	ConnId() string

	// Called to handle the next message from the input stream on the ws conn.
	HandleMessage(msg I) error

	// Called to handle or suppress an error
	OnError(err error) error

	// Called when the connection closes.
	OnClose()

	// Called when data has not been received within the PongPeriod.
	OnTimeout() bool
}
