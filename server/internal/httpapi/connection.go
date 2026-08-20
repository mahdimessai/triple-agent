package httpapi

import (
	"errors"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// connection owns the WebSocket writer side. The handler goroutine is the
// single reader; this writer loop is the single writer. The bounded queue is
// intentional backpressure: one slow browser must never block a room actor.
type connection struct {
	ws        *websocket.Conn
	out       chan any
	done      chan struct{}
	closeOnce sync.Once
}

func newConnection(ws *websocket.Conn) *connection {
	ws.SetReadLimit(readLimit)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(pongWait)) })
	return &connection{ws: ws, out: make(chan any, 16), done: make(chan struct{})}
}

func (c *connection) enqueue(value any) error {
	select {
	case <-c.done:
		return errors.New("connection closed")
	case c.out <- value:
		return nil
	default:
		c.close()
		return errors.New("connection is too slow")
	}
}

func (c *connection) writeLoop() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-c.done:
			return
		case message := <-c.out:
			if err := writeWebSocketJSON(c.ws, message); err != nil {
				c.close()
				return
			}
		case <-ticker.C:
			if err := c.ws.SetWriteDeadline(time.Now().Add(writeWait)); err != nil || c.ws.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)) != nil {
				c.close()
				return
			}
		}
	}
}

func (c *connection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}
