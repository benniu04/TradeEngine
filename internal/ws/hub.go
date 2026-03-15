package ws

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

// WSMessage is the envelope sent to connected clients.
type WSMessage struct {
	Type string      `json:"type"` // "order_update", "trade", "balance_update"
	Data interface{} `json:"data"`
}

// Client represents a single WebSocket connection.
type Client struct {
	UserID uuid.UUID
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub manages all active WebSocket connections, grouped by user ID.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]bool
}

func NewHub() *Hub {
	return &Hub{
		clients: make(map[uuid.UUID]map[*Client]bool),
	}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = make(map[*Client]bool)
	}
	h.clients[c.UserID][c] = true
	log.Printf("ws: client registered for user %s (%d total)", c.UserID, len(h.clients[c.UserID]))
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if conns, ok := h.clients[c.UserID]; ok {
		delete(conns, c)
		if len(conns) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	log.Printf("ws: client unregistered for user %s", c.UserID)
}

// SendToUser broadcasts a message to all connections for a given user.
func (h *Hub) SendToUser(userID uuid.UUID, msg WSMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}

	h.mu.RLock()
	conns := h.clients[userID]
	h.mu.RUnlock()

	for c := range conns {
		select {
		case c.Send <- data:
		default:
			// Client too slow, drop message
			log.Printf("ws: dropping message for slow client user=%s", userID)
		}
	}
}

// HandleWS is the HTTP handler that upgrades to a WebSocket connection.
// Expects ?user_id=<uuid> query parameter.
func (h *Hub) HandleWS(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.URL.Query().Get("user_id")
	if userIDStr == "" {
		http.Error(w, "missing user_id query parameter", http.StatusBadRequest)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		http.Error(w, "invalid user_id", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true, // allow any origin (dev mode)
	})
	if err != nil {
		log.Printf("ws: accept error: %v", err)
		return
	}

	client := &Client{
		UserID: userID,
		Conn:   conn,
		Send:   make(chan []byte, 64),
	}

	h.register(client)
	defer h.unregister(client)

	ctx := r.Context()

	// Writer goroutine: sends messages from Send channel to WebSocket
	go func() {
		for {
			select {
			case msg, ok := <-client.Send:
				if !ok {
					return
				}
				if err := conn.Write(ctx, websocket.MessageText, msg); err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Reader loop: keeps the connection alive, discards incoming messages
	for {
		_, _, err := conn.Read(ctx)
		if err != nil {
			break
		}
	}

	close(client.Send)
}
