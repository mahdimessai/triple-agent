package room

import "tripleagent/server/internal/game"

// ClientSession abstracts a connected client's transport lifecycle.
type ClientSession interface {
	ID() string
	Send(game.Projection) error
	Close()
}

// CallbackSession adapts functional closures to implement the ClientSession interface.
type CallbackSession struct {
	SessionID string
	SendFunc  func(game.Projection) error
	CloseFunc func()
}

func (s CallbackSession) ID() string {
	return s.SessionID
}

func (s CallbackSession) Send(projection game.Projection) error {
	if s.SendFunc == nil {
		return nil
	}
	return s.SendFunc(projection)
}

func (s CallbackSession) Close() {
	if s.CloseFunc != nil {
		s.CloseFunc()
	}
}
