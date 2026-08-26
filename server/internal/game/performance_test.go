package game

import (
	"bytes"
	"encoding/json"
	"strconv"
	"testing"
	"time"
)

func benchmarkState(playerCount int) State {
	state := NewLobby("p1", "P1")
	state.Settings.MinPlayers = playerCount
	state.Settings.MaxPlayers = playerCount
	state.RandomState = 1
	for index := 2; index <= playerCount; index++ {
		playerID := "p" + strconv.Itoa(index)
		var err error
		state, err = AddPlayer(state, playerID, playerID)
		if err != nil {
			panic(err)
		}
		state, err = Connect(state, playerID)
		if err != nil {
			panic(err)
		}
	}
	state.Phase = PhaseDiscussion
	deadline := time.Unix(100, 0).UTC().Add(time.Minute)
	state.DiscussionDeadline = &deadline
	return state
}

func BenchmarkCloneState(b *testing.B) {
	state := benchmarkState(9)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = cloneState(state)
	}
}

func BenchmarkApplyTransferHost(b *testing.B) {
	state := benchmarkState(9)
	command := Command{Kind: CommandTransferHost, TargetID: "p2"}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(state, "p1", command, time.Unix(100, 0).UTC())
	}
}

func BenchmarkApplyKickPlayer(b *testing.B) {
	state := benchmarkState(9)
	state.Phase = PhaseLobby
	command := Command{Kind: CommandKickPlayer, TargetID: "p9"}
	now := time.Unix(100, 0).UTC()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(state, "p1", command, now)
	}
}

func BenchmarkApplyVoteKick(b *testing.B) {
	state := benchmarkState(9)
	target := state.Players["p9"]
	target.Connected = false
	state.Players["p9"] = target
	command := Command{Kind: CommandVoteKick, TargetID: "p9"}
	now := time.Unix(100, 0).UTC()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(state, "p1", command, now)
	}
}

func BenchmarkApplyRematch(b *testing.B) {
	state := benchmarkState(9)
	state.Phase = PhaseEnd
	state.Winner = FactionService
	command := Command{Kind: CommandRematch}
	now := time.Unix(100, 0).UTC()
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_, _ = Apply(state, "p1", command, now)
	}
}

func BenchmarkProjectAndMarshal(b *testing.B) {
	state := benchmarkState(9)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		projection := Project("room", state, "p1")
		if _, err := json.Marshal(projection); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkResultsProjectionAndMarshal(b *testing.B) {
	state := benchmarkState(9)
	state.Phase = PhaseEnd
	state.Winner = FactionService
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		projection := Project("room", state, "p1")
		if _, err := json.Marshal(projection); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBroadcastProjectionAndMarshal(b *testing.B) {
	state := benchmarkState(9)
	public := PublicProjectionFor("room", state)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, playerID := range state.PlayerOrder {
			projection := ProjectWithPublic(state, playerID, public)
			if _, err := json.Marshal(projection); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func BenchmarkBroadcastProjectionAndEncodeBuffer(b *testing.B) {
	state := benchmarkState(9)
	public := PublicProjectionFor("room", state)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		for _, playerID := range state.PlayerOrder {
			buffer.Reset()
			projection := ProjectWithPublic(state, playerID, public)
			if err := encoder.Encode(projection); err != nil {
				b.Fatal(err)
			}
		}
	}
}
