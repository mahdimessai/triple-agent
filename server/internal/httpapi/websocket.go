package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"tripleagent/server/internal/game"

	"github.com/gorilla/websocket"
)

const (
	authTimeout = 5 * time.Second
	readLimit   = 16 << 10
	writeWait   = 10 * time.Second
	pongWait    = 60 * time.Second
	pingPeriod  = (pongWait * 9) / 10
)

type authMessage struct {
	Kind           string `json:"type"`
	ReconnectToken string `json:"reconnect_token"`
}

type commandMessage struct {
	Kind                   string   `json:"kind"`
	RequestID              string   `json:"request_id"`
	ExpectedVersion        uint64   `json:"expected_version"`
	OperationKind          string   `json:"operation_kind"`
	OperationEnabled       bool     `json:"operation_enabled"`
	RoleID                 string   `json:"role_id"`
	RoleEnabled            bool     `json:"role_enabled"`
	DiscussionTimerEnabled bool     `json:"discussion_timer_enabled"`
	DiscussionSeconds      int      `json:"discussion_seconds"`
	VirusCount             int      `json:"virus_count"`
	TargetID               string   `json:"target_id"`
	TargetIDs              []string `json:"target_ids"`
	Choice                 string   `json:"choice"`
}

type commandAck struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	OK        bool   `json:"ok"`
	Error     string `json:"error,omitempty"`
	Code      string `json:"code,omitempty"`
}

type sessionAuthenticatedResponse struct {
	Type string `json:"type"`
}

type sessionErrorResponse struct {
	Type   string `json:"type"`
	Status int    `json:"status"`
	Error  string `json:"error"`
	Code   string `json:"code,omitempty"`
}

type outboundKind uint8

const (
	outboundAuthenticated outboundKind = iota
	outboundProjection
	outboundCommandAck
)

type outboundMessage struct {
	kind       outboundKind
	projection game.Projection
	ack        commandAck
}

type wsSession struct {
	id        string
	conn      *connection
	firstSend bool
}

func (s *wsSession) ID() string {
	return s.id
}

func (s *wsSession) Send(projection game.Projection) error {
	if s.firstSend {
		s.firstSend = false
		if err := s.conn.enqueue(outboundMessage{kind: outboundAuthenticated}); err != nil {
			return err
		}
	}
	return s.conn.enqueue(outboundMessage{kind: outboundProjection, projection: projection})
}

func (s *wsSession) Close() {
	s.conn.close()
}

func (h *handler) websocket(w http.ResponseWriter, r *http.Request) {
	joinCode := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("join_code")))
	if joinCode == "" {
		writeHTTPError(w, newAPIError(http.StatusBadRequest, "session_identity_required", "Join code is required."))
		return
	}
	active, ok := h.roomManager.GetByCode(joinCode)
	if !ok {
		writeHTTPError(w, newAPIError(http.StatusNotFound, "room_not_found", "Lobby not found."))
		return
	}

	ws, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ws.SetReadLimit(readLimit)
	_ = ws.SetReadDeadline(time.Now().Add(authTimeout))

	var auth authMessage
	if err := readWebSocketJSON(ws, &auth); err != nil || auth.Kind != "room.auth" || strings.TrimSpace(auth.ReconnectToken) == "" {
		writeSessionError(ws, newAPIError(http.StatusUnauthorized, "unauthorized", "Authentication is required."))
		return
	}

	ws.SetReadLimit(readLimit)
	_ = ws.SetReadDeadline(time.Now().Add(pongWait))
	ws.SetPongHandler(func(string) error { return ws.SetReadDeadline(time.Now().Add(pongWait)) })

	conn := &connection{
		ws:   ws,
		out:  make(chan outboundMessage, 32),
		done: make(chan struct{}),
	}
	conn.encoder = json.NewEncoder(&conn.buffer)

	sessionID := "session_" + strconv.FormatUint(sessionSequence.Add(1), 10)
	sess := &wsSession{
		id:        sessionID,
		conn:      conn,
		firstSend: true,
	}

	playerID, _, attachErr := active.AttachSession(auth.ReconnectToken, sess)
	if attachErr != nil {
		writeSessionError(ws, sessionError(attachErr))
		return
	}

	go conn.writeLoop()
	defer func() {
		active.Detach(playerID, sessionID)
		conn.close()
	}()

	for {
		var message commandMessage
		if err := readWebSocketJSON(ws, &message); err != nil {
			return
		}
		if message.Kind == "room.resync" {
			projection, snapshotErr := active.Snapshot(playerID)
			if snapshotErr == nil {
				_ = conn.enqueue(outboundMessage{kind: outboundProjection, projection: projection})
			}
			continue
		}
		command := game.Command{
			Kind:                   game.CommandKind(message.Kind),
			OperationKind:          message.OperationKind,
			OperationEnabled:       message.OperationEnabled,
			RoleID:                 message.RoleID,
			RoleEnabled:            message.RoleEnabled,
			DiscussionTimerEnabled: message.DiscussionTimerEnabled,
			DiscussionSeconds:      message.DiscussionSeconds,
			VirusCount:             message.VirusCount,
			TargetID:               message.TargetID,
			TargetIDs:              message.TargetIDs,
			Choice:                 message.Choice,
		}
		commandErr := active.Command(playerID, sessionID, message.ExpectedVersion, command)
		code, errorMessage := commandError(commandErr)
		_ = conn.enqueue(outboundMessage{
			kind: outboundCommandAck,
			ack: commandAck{
				Type:      "command.ack",
				RequestID: message.RequestID,
				OK:        commandErr == nil,
				Error:     errorMessage,
				Code:      code,
			},
		})
	}
}

func readWebSocketJSON(conn *websocket.Conn, destination any) error {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(payload, destination)
}

func writeWebSocketJSON(conn *websocket.Conn, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeWebSocketPayload(conn, payload)
}

func writeSessionError(conn *websocket.Conn, apiErr *APIError) {
	_ = writeWebSocketJSON(conn, sessionErrorResponse{Type: "session.error", Status: apiErr.Status, Error: apiErr.Message, Code: apiErr.Code})
	_ = conn.Close()
}

type connection struct {
	ws        *websocket.Conn
	out       chan outboundMessage
	done      chan struct{}
	closeOnce sync.Once // used to ensure that the close sequence of a connection happens once.
	buffer    bytes.Buffer
	encoder   *json.Encoder
}

func (c *connection) writeJSON(value any) error {
	c.buffer.Reset()
	if err := c.encoder.Encode(value); err != nil {
		return err
	}
	return writeWebSocketPayload(c.ws, c.buffer.Bytes())
}

// enqueues an outbound message to a client connection.
// if the out channel is full already then this connection is slow af.
// close the connection if so (slow connection, may throttle the lobby more generally).
func (c *connection) enqueue(value outboundMessage) error {
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
			var err error
			switch message.kind {
			case outboundAuthenticated:
				err = c.writeJSON(sessionAuthenticatedResponse{Type: "session.authenticated"})
			case outboundProjection:
				err = c.writeJSON(message.projection)
			case outboundCommandAck:
				err = c.writeJSON(message.ack)
			default:
				c.close()
				return
			}
			if err != nil {
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

func writeWebSocketPayload(conn *websocket.Conn, payload []byte) error {
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func (c *connection) close() {
	c.closeOnce.Do(func() {
		close(c.done)
		_ = c.ws.Close()
	})
}

var sessionSequence atomic.Uint64
