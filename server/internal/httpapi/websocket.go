package httpapi

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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

func (h *handler) websocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimSpace(r.URL.Query().Get("room_id"))
	playerID := strings.TrimSpace(r.URL.Query().Get("player_id"))
	if roomID == "" || playerID == "" {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Room ID and player ID are required.", Code: "session_identity_required"})
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
		h.logger.Warn("websocket authentication rejected", "room_id", roomID, "player_id", playerID)
		writeSessionError(ws, http.StatusUnauthorized, "unauthorized", "Authentication is required.")
		return
	}
	active, ok := h.rooms.Get(roomID)
	if !ok {
		writeSessionError(ws, http.StatusGone, "room_gone", "Room is no longer available.")
		return
	}

	connection := newConnection(ws)
	sessionID := nextSessionID()
	firstSend := true
	sender := func(projection game.Projection) error {
		if firstSend {
			firstSend = false
			if err := connection.enqueue(map[string]string{"type": "session.authenticated"}); err != nil {
				return err
			}
		}
		return connection.enqueue(projection)
	}
	if err := active.Attach(playerID, auth.ReconnectToken, sessionID, sender, connection.close); err != nil {
		status, code, message := sessionError(err)
		if status >= http.StatusInternalServerError {
			h.logger.Error("room authentication failed", "room_id", roomID, "player_id", playerID, "error", err)
		}
		writeSessionError(ws, status, code, message)
		return
	}
	h.logger.Info("websocket session attached", "room_id", roomID, "player_id", playerID, "session_id", sessionID)
	go connection.writeLoop()
	defer func() {
		active.Detach(playerID, sessionID)
		connection.close()
		h.logger.Info("websocket session detached", "room_id", roomID, "player_id", playerID, "session_id", sessionID)
	}()

	for {
		var message commandMessage
		if err := readWebSocketJSON(ws, &message); err != nil {
			return
		}
		if message.Kind == "room.resync" {
			projection, snapshotErr := active.Snapshot(playerID)
			if snapshotErr == nil {
				_ = connection.enqueue(projection)
			}
			continue
		}
		if strings.TrimSpace(message.Kind) == "" || strings.TrimSpace(message.RequestID) == "" {
			_ = connection.enqueue(commandAck{Type: "command.ack", RequestID: message.RequestID, OK: false, Error: "command kind and request id are required", Code: "invalid_command"})
			continue
		}
		commandKind := game.CommandKind(message.Kind)
		if !game.IsKnownCommand(commandKind) {
			_ = connection.enqueue(commandAck{Type: "command.ack", RequestID: message.RequestID, OK: false, Error: "unknown command", Code: "unknown_command"})
			continue
		}

		command := game.Command{
			Kind: commandKind, OperationKind: message.OperationKind, OperationEnabled: message.OperationEnabled,
			RoleID: message.RoleID, RoleEnabled: message.RoleEnabled, DiscussionTimerEnabled: message.DiscussionTimerEnabled,
			DiscussionSeconds: message.DiscussionSeconds, VirusCount: message.VirusCount, TargetID: message.TargetID,
			TargetIDs: append([]string(nil), message.TargetIDs...), Choice: message.Choice,
		}
		commandErr := active.Command(playerID, sessionID, message.ExpectedVersion, command)
		code, errorMessage := commandError(commandErr)
		if commandErr != nil {
			attributes := []any{"room_id", roomID, "player_id", playerID, "session_id", sessionID, "request_id", message.RequestID, "command", message.Kind, "code", code}
			if code == "internal" {
				h.logger.Error("game command failed", append(attributes, "error", commandErr)...)
			} else {
				h.logger.Debug("game command rejected", attributes...)
			}
		}
		_ = connection.enqueue(commandAck{Type: "command.ack", RequestID: message.RequestID, OK: commandErr == nil, Error: errorMessage, Code: code})
	}
}

func readWebSocketJSON(conn *websocket.Conn, destination any) error {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	return decodeWebSocketJSON(payload, destination)
}

func decodeWebSocketJSON(payload []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return errInvalidJSON
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return errMultipleJSON
	}
	return nil
}

func writeWebSocketJSON(conn *websocket.Conn, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func writeSessionError(conn *websocket.Conn, status int, code, message string) {
	_ = writeWebSocketJSON(conn, map[string]any{"type": "session.error", "status": status, "error": message, "code": code})
	_ = conn.Close()
}

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

var sessionSequence atomic.Uint64

func nextSessionID() string { return "session_" + strconv.FormatUint(sessionSequence.Add(1), 10) }
