package gui

import (
	"encoding/json"
	"testing"

	"github.com/sonar-solutions/sonar-migration-tool/internal/wizard"
)

func TestToKVPairs(t *testing.T) {
	input := []wizard.KV{
		{Key: "URL", Value: "https://example.com"},
		{Key: "Token", Value: "********"},
	}
	result := ToKVPairs(input)
	if len(result) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(result))
	}
	if result[0].Key != "URL" || result[0].Value != "https://example.com" {
		t.Errorf("pair 0: got %+v", result[0])
	}
	if result[1].Key != "Token" || result[1].Value != "********" {
		t.Errorf("pair 1: got %+v", result[1])
	}
}

func TestToKVPairsEmpty(t *testing.T) {
	result := ToKVPairs(nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(result))
	}
}

func TestServerMessageJSON(t *testing.T) {
	msg := ServerMessage{
		Type:    TypePromptURL,
		ID:      "abc123",
		Message: "Server URL:",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ServerMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Type != TypePromptURL {
		t.Errorf("type: got %q, want %q", decoded.Type, TypePromptURL)
	}
	if decoded.ID != "abc123" {
		t.Errorf("id: got %q, want %q", decoded.ID, "abc123")
	}
	if decoded.Message != "Server URL:" {
		t.Errorf("message: got %q", decoded.Message)
	}
}

func TestClientMessageJSON(t *testing.T) {
	msg := ClientMessage{
		Type:  TypePromptResponse,
		ID:    "abc123",
		Value: "https://sonar.example.com/",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ClientMessage
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Type != TypePromptResponse {
		t.Errorf("type: got %q", decoded.Type)
	}
	if decoded.Value != "https://sonar.example.com/" {
		t.Errorf("value: got %v", decoded.Value)
	}
}

func TestClientMessageBoolValueJSON(t *testing.T) {
	raw := `{"type":"prompt_response","id":"x","value":true}`
	var msg ClientMessage
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatal(err)
	}
	if msg.Value != true {
		t.Errorf("expected true, got %v (%T)", msg.Value, msg.Value)
	}
}

func TestServerMessageOmitsEmptyFields(t *testing.T) {
	msg := ServerMessage{Type: TypeDisplayWelcome}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	json.Unmarshal(data, &m)

	// These fields should be omitted (empty).
	for _, key := range []string{"id", "message", "title", "details"} {
		if _, ok := m[key]; ok {
			t.Errorf("field %q should be omitted for empty value", key)
		}
	}
	// "error" should always be present (not omitempty) — null is valid.
	if _, ok := m["error"]; !ok {
		t.Error("field \"error\" should always be present")
	}
}
