package transport

import (
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/bytedance/sonic"
	"github.com/gorilla/websocket"
)

var (
	httpJSON      = sonic.ConfigStd
	websocketJSON = sonic.ConfigDefault
)

func readWebSocketJSON(conn *websocket.Conn, destination any) error {
	_, payload, err := conn.ReadMessage()
	if err != nil {
		return err
	}
	return websocketJSON.Unmarshal(payload, destination)
}

func writeWebSocketJSON(conn *websocket.Conn, value any) error {
	payload, err := websocketJSON.Marshal(value)
	if err != nil {
		return err
	}
	if err := conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
		return err
	}
	return conn.WriteMessage(websocket.TextMessage, payload)
}

func writeSessionError(conn *websocket.Conn, err error) {
	status := statusFromSessionError(err)
	message := fmsg.GetIssue(err)
	if message == "" {
		message = messageFromStatus(status)
	}
	_ = writeWebSocketJSON(conn, map[string]any{"type": "session.error", "status": status, "error": message})
	_ = conn.Close()
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = httpJSON.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	decoder := httpJSON.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fault.Wrap(err,
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("invalid JSON", "Invalid JSON."),
		)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return fault.Wrap(errors.New("request must contain one JSON value"),
			ftag.With(ftag.InvalidArgument),
			fmsg.WithDesc("request must contain one JSON value", "Request must contain one JSON value."),
		)
	}
	return nil
}
