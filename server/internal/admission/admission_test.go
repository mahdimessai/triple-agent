package admission

import (
	"errors"
	"testing"
)

func TestReserveClaimsTheCodeAndRegistersTheHostCredential(t *testing.T) {
	store := NewStore()
	roomID := "room_test"
	hostID := "player_host"
	token := "secret-token"

	if err := store.Reserve(roomID, "ABC123", hostID, token); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateReconnectToken(roomID, hostID, token); err != nil {
		t.Fatal(err)
	}
	if err := store.ValidateReconnectToken(roomID, hostID, "wrong-token"); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("wrong reconnect token check = %v, want ErrInvalidToken", err)
	}

	resolved, canonical, err := store.ResolveCode("abc123")
	if err != nil || resolved != roomID || canonical != "ABC123" {
		t.Fatalf("lookup by code = %s/%s, err = %v", resolved, canonical, err)
	}

	if err := store.Reserve("room_other", "abc123", "someone", "another-token"); !errors.Is(err, ErrJoinCodeTaken) {
		t.Fatalf("duplicate code reservation = %v, want ErrJoinCodeTaken", err)
	}
}

func TestGrantAndRevokePlayerCredentials(t *testing.T) {
	store := NewStore()
	roomID := "room_test"
	hostID := "player_host"
	playerID := "player_two"
	token := "player-token"

	if err := store.Reserve(roomID, "ABC123", hostID, "host-token"); err != nil {
		t.Fatal(err)
	}

	if err := store.Grant(roomID, playerID, token); err != nil {
		t.Fatal(err)
	}

	if err := store.ValidateReconnectToken(roomID, playerID, token); err != nil {
		t.Fatal("granted player credential did not validate:", err)
	}

	store.Revoke(roomID, playerID)

	if err := store.ValidateReconnectToken(roomID, playerID, token); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("revoked credential = %v, want ErrInvalidToken", err)
	}
}

func TestReleaseFreesTheRoomAndItsCode(t *testing.T) {
	store := NewStore()
	roomID := "room_release"

	if err := store.Reserve(roomID, "GONE01", "host", "host-token"); err != nil {
		t.Fatal(err)
	}

	store.Release(roomID)

	if _, _, err := store.ResolveCode("GONE01"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("released code lookup = %v, want ErrRoomNotFound", err)
	}
	if err := store.ValidateReconnectToken(roomID, "host", "host-token"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("released room credential = %v, want ErrRoomNotFound", err)
	}

	// Code can now be reused
	if err := store.Reserve("room_reused", "GONE01", "new-host", "new-token"); err != nil {
		t.Fatalf("reusing a released join code: %v", err)
	}
}
