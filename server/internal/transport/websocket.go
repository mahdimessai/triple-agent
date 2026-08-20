package transport

import (
	"net/http"
	"strings"
	"time"

	"tripleagent/server/internal/domain"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/gorilla/websocket"
)

// WebSocket authenticates one room session and translates frames to room commands.
func (h *Handler) WebSocket(w http.ResponseWriter, r *http.Request) {
	roomID := strings.TrimSpace(r.URL.Query().Get("room_id"))
	playerID := strings.TrimSpace(r.URL.Query().Get("player_id"))
	if roomID == "" || playerID == "" {
		writeError(w, fault.New("room_id and player_id are required",
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("room_id and player_id are required", "Room ID and player ID are required."),
		))
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(*http.Request) bool {
			return true
		},
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	ws.SetReadLimit(readLimit)
	_ = ws.SetReadDeadline(time.Now().Add(authTimeout))

	var auth authMessage
	if err := readWebSocketJSON(ws, &auth); err != nil || auth.Kind != "room.auth" || strings.TrimSpace(auth.ReconnectToken) == "" {
		writeSessionError(ws, fault.New("authentication is required",
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("authentication is required", "Authentication is required."),
		))
		return
	}
	activeRoom, err := h.sessions.Authenticate(roomID, playerID, auth.ReconnectToken)
	if err != nil {
		writeSessionError(ws, err)
		return
	}

	connection := newConnection(ws)
	go connection.writeLoop()
	if err := connection.sendJSON(map[string]string{"type": "session.authenticated"}); err != nil {
		connection.close()
		return
	}

	sessionID := nextSessionID()
	session, err := h.sessions.Attach(activeRoom, roomID, playerID, sessionID, connection.sendProjection, connection.close)
	if err != nil {
		connection.close()
		return
	}
	defer func() {
		h.sessions.Detach(session)
		connection.close()
	}()

	for {
		var message commandMessage
		if err := readWebSocketJSON(ws, &message); err != nil {
			return
		}
		if message.Kind == "room.resync" {
			projection, snapshotErr := activeRoom.Snapshot(playerID)
			if snapshotErr == nil {
				_ = connection.sendProjection(projection)
			}
			continue
		}

		command := message.commandFor(playerID)
		projection, replayed, commandErr := activeRoom.SubmitForSession(sessionID, command)
		if replayed && commandErr == nil {
			_ = connection.sendProjection(projection)
		}
		errorMessage := ""
		if commandErr != nil {
			errorMessage = commandErr.Error()
		}
		_ = connection.sendJSON(map[string]any{
			"type":       "command.ack",
			"request_id": command.RequestID,
			"ok":         commandErr == nil,
			"error":      errorMessage,
		})
	}
}

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

// commandFor is the trust boundary between the wire protocol and the domain.
// ActorID is server-derived and can never be supplied by a client frame.
func (m commandMessage) commandFor(playerID string) domain.Command {
	return domain.Command{
		RequestID:              m.RequestID,
		ActorID:                playerID,
		ExpectedVersion:        m.ExpectedVersion,
		Kind:                   domain.CommandKind(m.Kind),
		OperationKind:          m.OperationKind,
		OperationEnabled:       m.OperationEnabled,
		RoleID:                 m.RoleID,
		RoleEnabled:            m.RoleEnabled,
		DiscussionTimerEnabled: m.DiscussionTimerEnabled,
		DiscussionSeconds:      m.DiscussionSeconds,
		VirusCount:             m.VirusCount,
		TargetID:               m.TargetID,
		TargetIDs:              append([]string(nil), m.TargetIDs...),
		Choice:                 m.Choice,
	}
}
