package gui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// dialTestWS connects a test WebSocket client to a hub served by httptest.
func dialTestWS(t *testing.T, hub *Hub) (*websocket.Conn, func()) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(hub.ServeWS))
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial: %v", err)
	}
	return conn, func() { conn.Close(); srv.Close() }
}

func TestHubSendToConnectedClient(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	conn, cleanup := dialTestWS(t, hub)
	defer cleanup()

	// Give the read loop time to start.
	time.Sleep(20 * time.Millisecond)

	hub.Send(ServerMessage{Type: TypeDisplayMessage, Message: "hello"})

	conn.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}

	var msg ServerMessage
	json.Unmarshal(data, &msg)
	if msg.Type != TypeDisplayMessage || msg.Message != "hello" {
		t.Errorf("got %+v", msg)
	}
}

func TestHubSendNoConnection(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	// Should not panic with no connection.
	hub.Send(ServerMessage{Type: TypeDisplayMessage, Message: "ignored"})
}

func TestHubDispatchStartWizard(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	called := make(chan struct{}, 1)
	hub.OnStartWizard = func() { called <- struct{}{} }

	conn, cleanup := dialTestWS(t, hub)
	defer cleanup()

	time.Sleep(20 * time.Millisecond)
	msg, _ := json.Marshal(ClientMessage{Type: TypeStartWizard})
	conn.WriteMessage(websocket.TextMessage, msg)

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("OnStartWizard not called")
	}
}

func TestHubDispatchCancelWizard(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	called := make(chan struct{}, 1)
	hub.OnCancelWizard = func() { called <- struct{}{} }

	conn, cleanup := dialTestWS(t, hub)
	defer cleanup()

	time.Sleep(20 * time.Millisecond)
	msg, _ := json.Marshal(ClientMessage{Type: TypeCancelWizard})
	conn.WriteMessage(websocket.TextMessage, msg)

	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("OnCancelWizard not called")
	}
}

func TestHubDispatchPromptResponse(t *testing.T) {
	ctx := context.Background()
	var sentMsgs []ServerMessage
	sendFn := func(msg ServerMessage) { sentMsgs = append(sentMsgs, msg) }
	wp := NewWebPrompter(ctx, sendFn)
	hub := NewHub(wp)

	conn, cleanup := dialTestWS(t, hub)
	defer cleanup()

	time.Sleep(20 * time.Millisecond)

	// Start a prompt in a goroutine.
	done := make(chan string)
	go func() {
		result, _ := wp.PromptText("Name:", "")
		done <- result
	}()

	// Wait for prompt to be sent, get the ID.
	time.Sleep(30 * time.Millisecond)
	if len(sentMsgs) == 0 {
		t.Fatal("no prompt sent")
	}
	promptID := sentMsgs[0].ID

	// Respond via WebSocket.
	resp, _ := json.Marshal(ClientMessage{
		Type:  TypePromptResponse,
		ID:    promptID,
		Value: "Keith",
	})
	conn.WriteMessage(websocket.TextMessage, resp)

	select {
	case result := <-done:
		if result != "Keith" {
			t.Errorf("got %q, want Keith", result)
		}
	case <-time.After(time.Second):
		t.Fatal("prompt did not return")
	}
}

func TestHubReplacesConnection(t *testing.T) {
	wp := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp)

	conn1, cleanup1 := dialTestWS(t, hub)
	defer cleanup1()

	time.Sleep(20 * time.Millisecond)

	// Connect a second client — first should be replaced.
	conn2, cleanup2 := dialTestWS(t, hub)
	defer cleanup2()

	time.Sleep(20 * time.Millisecond)

	// Send a message — only conn2 should receive it.
	hub.Send(ServerMessage{Type: TypeDisplayMessage, Message: "to conn2"})

	conn2.SetReadDeadline(time.Now().Add(time.Second))
	_, data, err := conn2.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	var msg ServerMessage
	json.Unmarshal(data, &msg)
	if msg.Message != "to conn2" {
		t.Errorf("conn2 got: %+v", msg)
	}

	// conn1 should be closed — reading should fail.
	conn1.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	_, _, err = conn1.ReadMessage()
	if err == nil {
		t.Error("conn1 should be closed after replacement")
	}
}

func TestHubSetPrompter(t *testing.T) {
	wp1 := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub := NewHub(wp1)

	wp2 := NewWebPrompter(context.Background(), func(ServerMessage) {})
	hub.SetPrompter(wp2)

	hub.mu.Lock()
	got := hub.prompter
	hub.mu.Unlock()

	if got != wp2 {
		t.Error("SetPrompter did not replace the prompter")
	}
}
