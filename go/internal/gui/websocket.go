package gui

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // local-only server
}

// Hub manages a single WebSocket connection (single-user tool).
type Hub struct {
	mu       sync.Mutex
	conn     *websocket.Conn
	prompter *WebPrompter

	OnStartWizard  func()
	OnCancelWizard func()
}

// NewHub creates a Hub that routes prompt responses to the given WebPrompter.
func NewHub(prompter *WebPrompter) *Hub {
	return &Hub{prompter: prompter}
}

// SetPrompter replaces the active WebPrompter (used when starting a new wizard run).
func (h *Hub) SetPrompter(wp *WebPrompter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.prompter = wp
}

// Send marshals and writes a ServerMessage to the active connection.
func (h *Hub) Send(msg ServerMessage) {
	h.mu.Lock()
	conn := h.conn
	h.mu.Unlock()

	if conn == nil {
		return
	}

	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("gui: marshal error: %v", err)
		return
	}

	h.mu.Lock()
	err = conn.WriteMessage(websocket.TextMessage, data)
	h.mu.Unlock()

	if err != nil {
		log.Printf("gui: write error: %v", err)
	}
}

// ServeWS handles WebSocket upgrade and enters the read loop.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("gui: upgrade error: %v", err)
		return
	}

	// Replace any existing connection (single-user).
	h.mu.Lock()
	if h.conn != nil {
		h.conn.Close()
	}
	h.conn = conn
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		if h.conn == conn {
			h.conn = nil
		}
		h.mu.Unlock()
		conn.Close()
	}()

	h.readLoop(conn)
}

func (h *Hub) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("gui: read error: %v", err)
			}
			return
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			log.Printf("gui: unmarshal error: %v", err)
			continue
		}

		h.dispatch(msg)
	}
}

func (h *Hub) dispatch(msg ClientMessage) {
	switch msg.Type {
	case TypePromptResponse:
		h.mu.Lock()
		p := h.prompter
		h.mu.Unlock()
		if p != nil {
			p.HandleResponse(msg)
		}
	case TypeStartWizard:
		if h.OnStartWizard != nil {
			h.OnStartWizard()
		}
	case TypeCancelWizard:
		if h.OnCancelWizard != nil {
			h.OnCancelWizard()
		}
	default:
		log.Printf("gui: unknown message type: %s", msg.Type)
	}
}
