package http

import "time"

type BiDirStreamConfig struct {
	PingPeriod time.Duration
	PongPeriod time.Duration
}

func DefaultBiDirStreamConfig() *BiDirStreamConfig {
	return &BiDirStreamConfig{
		PingPeriod: time.Second * 30,
		PongPeriod: time.Second * 300,
	}
}

type BiDirStreamConn[I any] interface {
	/**
	 * Called to send the next ping message.
	 */
	SendPing() error

	// Optional Name of the connection
	Name() string

	// Optional connection id
	ConnId() string

	/**
	 * Called to handle the next message from the input stream on the ws conn.
	 */
	HandleMessage(msg I) error

	/**
	 * Called to handle or suppress an error
	 */
	OnError(err error) error

	/**
	 * Called when the connection closes.
	 */
	OnClose()
	OnTimeout() bool
}
