package transport

import (
	"errors"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"tripleagent/server/internal/domain"

	"github.com/gorilla/websocket"
)

// connection owns one authenticated WebSocket's write queue, heartbeat, and close state.
type connection struct {
	// ws is the underlying WebSocket connection.
	ws *websocket.Conn
	// out buffers outbound projections, acknowledgements, and heartbeats.
	out chan any
	// done signals the writer and producers that this connection is closed.
	done chan struct{}
	// closeOnce makes connection shutdown safe across competing goroutines.
	closeOnce sync.Once
}

func newConnection(ws *websocket.Conn) *connection {
	ws.SetReadLimit(readLimit)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error {
		return ws.SetReadDeadline(time.Now().Add(pongWait))
	})
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

func (c *connection) sendProjection(projection domain.Projection) error {
	return c.enqueue(projection)
}

func (c *connection) sendJSON(value any) error {
	return c.enqueue(value)
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

var sessionSequence atomic.Uint64

func nextSessionID() string {
	return "session_" + strconv.FormatUint(sessionSequence.Add(1), 10)
}
