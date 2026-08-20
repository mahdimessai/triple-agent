package admission

import (
	"errors"
	"testing"
)

func TestReserveClaimsTheCode(t *testing.T) {
	store := NewStore()
	roomID := "room_test"

	if err := store.Reserve(roomID, "ABC123"); err != nil {
		t.Fatal(err)
	}

	resolved, canonical, err := store.ResolveCode("abc123")
	if err != nil || resolved != roomID || canonical != "ABC123" {
		t.Fatalf("lookup by code = %s/%s, err = %v", resolved, canonical, err)
	}

	if err := store.Reserve("room_other", "abc123"); !errors.Is(err, ErrJoinCodeTaken) {
		t.Fatalf("duplicate code reservation = %v, want ErrJoinCodeTaken", err)
	}
}

func TestReleaseFreesTheRoomAndItsCode(t *testing.T) {
	store := NewStore()
	roomID := "room_release"

	if err := store.Reserve(roomID, "GONE01"); err != nil {
		t.Fatal(err)
	}

	store.Release(roomID)

	if _, _, err := store.ResolveCode("GONE01"); !errors.Is(err, ErrRoomNotFound) {
		t.Fatalf("released code lookup = %v, want ErrRoomNotFound", err)
	}

	if err := store.Reserve("room_reused", "GONE01"); err != nil {
		t.Fatalf("reusing a released join code: %v", err)
	}
}
