package room

import (
	"strconv"
	"testing"

	"tripleagent/server/internal/domain"
)

func benchmarkRoomState(playerCount int) domain.GameState {
	state := domain.NewLobby("room-benchmark", "player-0", "Player 0", domain.DefaultRoomSettings())
	for i := 1; i < playerCount; i++ {
		id := string("player-" + strconv.Itoa(i))
		if err := state.AddPlayer(id, "Player "+strconv.Itoa(i)); err != nil {
			panic(err)
		}
	}
	state.Phase = domain.PhaseDiscussion
	state.Version = 42
	return state
}

func broadcastWithoutSharedPublic(state domain.GameState, sessions map[string]roomSession) {
	for playerID, session := range sessions {
		_ = session.sender(domain.Project(state, playerID))
	}
}

func BenchmarkBroadcastSevenPlayerRoomBefore(b *testing.B) {
	state := benchmarkRoomState(7)
	sessions := benchmarkRoomSessions(state)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		broadcastWithoutSharedPublic(state, sessions)
	}
}

func BenchmarkBroadcastSevenPlayerRoomAfter(b *testing.B) {
	state := benchmarkRoomState(7)
	sessions := benchmarkRoomSessions(state)
	r := &Room{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.broadcast(&state, sessions)
	}
}

func benchmarkRoomSessions(state domain.GameState) map[string]roomSession {
	sessions := make(map[string]roomSession, len(state.PlayerOrder))
	for _, playerID := range state.PlayerOrder {
		sessions[playerID] = roomSession{sender: func(domain.Projection) error { return nil }}
	}
	return sessions
}
