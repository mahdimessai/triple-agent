package app

import (
	"errors"

	"tripleagent/server/internal/admission"
	"tripleagent/server/internal/domain"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

// wrapLobbyError translates collaborator failures at the application boundary,
// so transport only needs to understand fault tags.
func wrapLobbyError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrRoomInactive):
		return fault.Wrap(err,
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("room is no longer active", "Room is no longer active."),
		)
	case errors.Is(err, ErrLobbyStarted), errors.Is(err, domain.ErrNotAllowed):
		return fault.Wrap(err,
			ftag.With(ftag.AlreadyExists),
			fmsg.WithDesc("lobby has already started", "Lobby has already started."),
		)
	case errors.Is(err, admission.ErrRoomNotFound):
		return fault.Wrap(err,
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("lobby not found", "Lobby not found."),
		)
	case errors.Is(err, admission.ErrInvalidToken):
		return fault.Wrap(err,
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("invalid reconnect token", "Invalid reconnect token."),
		)
	case errors.Is(err, domain.ErrRoomFull):
		return fault.Wrap(err,
			ftag.With(ftag.AlreadyExists),
			fmsg.WithDesc("room is full", "Room is full."),
		)
	case errors.Is(err, admission.ErrJoinCodeTaken):
		return fault.Wrap(err,
			ftag.With(ftag.AlreadyExists),
			fmsg.WithDesc("join code is already in use", "Join code is already in use."),
		)
	default:
		return fault.Wrap(err, ftag.With(ftag.Internal), fmsg.With("lobby operation unavailable"))
	}
}

func wrapSessionError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrRoomInactive):
		return fault.Wrap(err,
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("room is no longer active", "Room is no longer active."),
		)
	case errors.Is(err, admission.ErrRoomNotFound):
		return fault.Wrap(err,
			ftag.With(ftag.NotFound),
			fmsg.WithDesc("room is no longer available", "Room is no longer available."),
		)
	case errors.Is(err, admission.ErrInvalidToken):
		return fault.Wrap(err,
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("invalid reconnect token", "Invalid reconnect token."),
		)
	case errors.Is(err, domain.ErrPlayerNotInRoom):
		return fault.Wrap(err,
			ftag.With(ftag.Unauthenticated),
			fmsg.WithDesc("player is no longer in room", "This player is no longer seated in the room."),
		)
	default:
		return fault.Wrap(err, ftag.With(ftag.Internal), fmsg.With("room authentication unavailable"))
	}
}
