package transport

import (
	"net/http"

	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

func writeError(w http.ResponseWriter, err error) {
	status := statusFromError(err)
	message := fmsg.GetIssue(err)
	if message == "" {
		message = messageFromStatus(status)
	}
	writeJSON(w, status, map[string]string{"error": message})
}

func statusFromError(err error) int {
	switch ftag.Get(err) {
	case ftag.NotFound:
		return http.StatusNotFound
	case ftag.InvalidArgument:
		return http.StatusBadRequest
	case ftag.AlreadyExists:
		return http.StatusConflict
	case ftag.PermissionDenied:
		return http.StatusForbidden
	case ftag.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func statusFromSessionError(err error) int {
	switch ftag.Get(err) {
	case ftag.NotFound:
		// The client treats 410 as terminal and destroys the session. An
		// authenticated session whose room is gone cannot recover by retrying.
		return http.StatusGone
	case ftag.Unauthenticated:
		return http.StatusUnauthorized
	default:
		return http.StatusInternalServerError
	}
}

func messageFromStatus(status int) string {
	switch status {
	case http.StatusNotFound:
		return "resource not found"
	case http.StatusBadRequest:
		return "invalid request"
	case http.StatusConflict:
		return "resource already exists"
	case http.StatusForbidden:
		return "permission denied"
	case http.StatusUnauthorized:
		return "authentication required"
	case http.StatusGone:
		return "room is no longer available"
	default:
		return "internal server error"
	}
}
