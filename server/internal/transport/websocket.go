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
	roomID := string(strings.TrimSpace(r.URL.Query().Get("room_id")))
	playerID := string(strings.TrimSpace(r.URL.Query().Get("player_id")))
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
		CheckOrigin: func(request *http.Request) bool {
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
		command := domain.Command{
			RequestID: message.RequestID, ActorID: playerID, ExpectedVersion: message.ExpectedVersion,
			Kind: domain.CommandKind(message.Kind), OperationKind: message.OperationKind, OperationEnabled: message.OperationEnabled, RoleID: message.RoleID, RoleEnabled: message.RoleEnabled, DiscussionTimerEnabled: message.DiscussionTimerEnabled, DiscussionSeconds: message.DiscussionSeconds, VirusCount: message.VirusCount, TargetID: string(message.TargetID), TargetIDs: toIDs(message.TargetIDs), Choice: message.Choice,
		}
		projection, replayed, commandErr := activeRoom.SubmitForSession(sessionID, command)
		if replayed && commandErr == nil {
			_ = connection.sendProjection(projection)
		}
		errorMessage := ""
		if commandErr != nil {
			errorMessage = commandErr.Error()
		}
		_ = connection.sendJSON(map[string]any{
			"type": "command.ack", "request_id": command.RequestID, "ok": commandErr == nil, "error": errorMessage,
		})
	}
}

type authMessage struct {
	// Kind must be room.auth for the first frame on a WebSocket connection.
	Kind string `json:"type"`
	// ReconnectToken authenticates the player identified by the query parameters.
	ReconnectToken string `json:"reconnect_token"`
}

type commandMessage struct {
	// Kind selects the domain command or the room.resync control message.
	Kind string `json:"kind"`
	// RequestID makes command retries idempotent inside the room actor.
	RequestID string `json:"request_id"`
	// ExpectedVersion prevents a client from applying a command to stale state.
	ExpectedVersion uint64 `json:"expected_version"`
	// OperationKind identifies the operation being configured or resolved.
	OperationKind string `json:"operation_kind"`
	// OperationEnabled changes whether an operation is in the lobby pool.
	OperationEnabled bool `json:"operation_enabled"`
	// RoleID identifies the special role being configured.
	RoleID string `json:"role_id"`
	// RoleEnabled changes whether a special role is in the lobby pool.
	RoleEnabled bool `json:"role_enabled"`
	// DiscussionTimerEnabled controls the lobby's discussion timer setting.
	DiscussionTimerEnabled bool `json:"discussion_timer_enabled"`
	// DiscussionSeconds sets how many seconds the discussion runs for.
	DiscussionSeconds int `json:"discussion_seconds"`
	// VirusCount controls how many players begin on the VIRUS faction.
	VirusCount int `json:"virus_count"`
	// TargetID is the single target selected by a command.
	TargetID string `json:"target_id"`
	// TargetIDs contains the targets for operations that select multiple players.
	TargetIDs []string `json:"target_ids"`
	// Choice carries a command-specific choice such as STAY or DEFECT.
	Choice string `json:"choice"`
}

func toIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	ids := make([]string, 0, len(values))
	for _, value := range values {
		ids = append(ids, string(value))
	}
	return ids
}
