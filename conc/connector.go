package conc

import (
	"log"
	"time"
)

type Connector[M any] struct {
	connect func() error
	read    ReaderFunc[M]
	// Called when a new message is received
	OnMessage func(msg Message[M]) error

	// Called when the connection is closed and quit (wont be called on reconnects)
	OnClose func()

	controlChan chan string
}

/**
 * Encapsulates a connection object which connects and continuously
 * reads out of it.  This connection also has other facilities like
 * reconnecting on failures or closes (by recalling the connect
 * method) with retries and backoffs etc.
 */
func NewConnector[M any](connect func() error, read ReaderFunc[M]) *Connector[M] {
	conn := &Connector[M]{
		connect: connect,
		read:    read,
	}
	return conn
}

func (c *Connector[M]) Stop() {
	c.controlChan <- "stop"
}

func (c *Connector[M]) Start() {
	// connect first
	connReader := NewReader(c.read)
	ticker := time.NewTicker(time.Second * 10)
	defer func() {
		connReader.Stop()
		ticker.Stop()
		if c.OnClose != nil {
			c.OnClose()
		}
	}()
	for {
		err := c.connect()
		if err != nil {
			log.Println("Connect error: ", err)
			// get into retry mode with our backoff semantics etc
			continue
		}
		select {
		case <-ticker.C:
			// check for a ping
			break
		case <-c.controlChan:
			// stopped
			return
		case msg := <-connReader.RecvChan():
			if msg.Error != nil {
				log.Print("Error reading client message: ", msg.Error)
				// may have closed so break out of this and go back to
				// our reconnect/backoff/etc logic
				break
			} else {
				c.OnMessage(msg)
			}
			break
		}
	}
}
