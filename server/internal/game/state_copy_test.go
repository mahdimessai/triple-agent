package game

import (
	"testing"
	"time"
)

func TestCopyOnWriteTransitionsPreservePreviousState(t *testing.T) {
	state := testLobby(t, 3)
	now := time.Unix(100, 0).UTC()

	transferred, err := Apply(state, "p1", Command{Kind: CommandTransferHost, TargetID: "p2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.HostID != "p1" || transferred.HostID != "p2" {
		t.Fatalf("host transition mutated the source: source=%q next=%q", state.HostID, transferred.HostID)
	}

	updatedSettings, err := Apply(state, "p1", Command{Kind: CommandSetOperationEnabled, OperationKind: "Share", OperationEnabled: false}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Settings.EnabledOperations["Share"] || updatedSettings.Settings.EnabledOperations["Share"] {
		t.Fatal("operation settings were aliased between states")
	}

	readyState, err := Apply(state, "p1", Command{Kind: CommandSetReady}, now)
	if err != nil {
		t.Fatal(err)
	}
	if state.Players["p1"].Ready || !readyState.Players["p1"].Ready {
		t.Fatal("player map was aliased between states")
	}

	discussion := state
	discussion.Phase = PhaseDiscussion
	discussion.DiscussionAcks = map[string]bool{}
	acknowledged, err := Apply(discussion, "p1", Command{Kind: CommandAdvanceDiscussion}, now)
	if err != nil {
		t.Fatal(err)
	}
	if discussion.DiscussionAcks["p1"] || !acknowledged.DiscussionAcks["p1"] {
		t.Fatal("discussion acknowledgements were aliased between states")
	}

	voting := state
	voting.Phase = PhaseVoteInput
	voting.Vote.Submitted = map[string]string{}
	voted, err := Apply(voting, "p1", Command{Kind: CommandSubmitVote, TargetID: "p2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := voting.Vote.Submitted["p1"]; exists {
		t.Fatal("vote submissions were aliased between states")
	}
	if voted.Vote.Submitted["p1"] != "p2" {
		t.Fatal("vote was not committed to the next state")
	}
}

func TestPlayerRemovalAndRematchTransitionsPreservePreviousState(t *testing.T) {
	state := testLobby(t, 3)
	now := time.Unix(100, 0).UTC()

	kicked, err := Apply(state, "p1", Command{Kind: CommandKickPlayer, TargetID: "p2"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !HasPlayer(state, "p2") || HasPlayer(kicked, "p2") {
		t.Fatal("kick transition aliased player ownership")
	}

	voteState := copyPlayers(state)
	target := voteState.Players["p3"]
	target.Connected = false
	voteState.Players["p3"] = target
	voted, err := Apply(voteState, "p1", Command{Kind: CommandVoteKick, TargetID: "p3"}, now)
	if err != nil {
		t.Fatal(err)
	}
	if voteState.VoteKicks != nil || !voted.VoteKicks["p3"]["p1"] {
		t.Fatal("vote-kick transition aliased vote ownership")
	}

	results := copyPlayers(state)
	results.Phase = PhaseEnd
	results.Players["p1"] = func() Player {
		player := results.Players["p1"]
		player.Ready = true
		return player
	}()
	rematch, err := Apply(results, "p1", Command{Kind: CommandRematch}, now)
	if err != nil {
		t.Fatal(err)
	}
	if !results.Players["p1"].Ready || rematch.Players["p1"].Ready {
		t.Fatal("rematch transition aliased player ownership")
	}
}
