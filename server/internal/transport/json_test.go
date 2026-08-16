package transport

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

type jsonTestRequest struct {
	Name string `json:"name"`
}

func TestDecodeJSONRejectsUnknownFields(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Agent A","extra":true}`))

	var destination jsonTestRequest
	if err := decodeJSON(recorder, request, &destination); err == nil {
		t.Fatal("decodeJSON accepted an unknown field")
	}
}

func TestDecodeJSONRejectsTrailingValues(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"Agent A"}{"name":"Agent B"}`))

	var destination jsonTestRequest
	if err := decodeJSON(recorder, request, &destination); err == nil {
		t.Fatal("decodeJSON accepted trailing JSON")
	}
}

func TestDecodeJSONRejectsOversizedBodies(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest("POST", "/", strings.NewReader(`{"name":"`+strings.Repeat("a", maxJSONBody)+`"}`))

	var destination jsonTestRequest
	if err := decodeJSON(recorder, request, &destination); err == nil {
		t.Fatal("decodeJSON accepted a body larger than the configured limit")
	}
}

func BenchmarkWebSocketJSONMarshalCommand(b *testing.B) {
	message := commandMessage{RequestID: "req-1", ExpectedVersion: 42, OperationKind: "choose", RoleID: "role-1", TargetID: "player-2"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := websocketJSON.Marshal(message); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkWebSocketJSONUnmarshalCommand(b *testing.B) {
	payload, err := websocketJSON.Marshal(commandMessage{RequestID: "req-1", ExpectedVersion: 42, OperationKind: "choose", RoleID: "role-1", TargetID: "player-2"})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var message commandMessage
		if err := websocketJSON.Unmarshal(payload, &message); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdJSONMarshalCommand(b *testing.B) {
	message := commandMessage{RequestID: "req-1", ExpectedVersion: 42, OperationKind: "choose", RoleID: "role-1", TargetID: "player-2"}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := json.Marshal(message); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStdJSONUnmarshalCommand(b *testing.B) {
	payload, err := json.Marshal(commandMessage{RequestID: "req-1", ExpectedVersion: 42, OperationKind: "choose", RoleID: "role-1", TargetID: "player-2"})
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var message commandMessage
		if err := json.Unmarshal(payload, &message); err != nil {
			b.Fatal(err)
		}
	}
}
