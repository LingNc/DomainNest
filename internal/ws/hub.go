package ws

import (
	"encoding/json"
	"log"
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages.
type Hub struct {
	mu      sync.RWMutex
	clients map[uint64]*Client // userID -> single active connection
}

// defaultHub is the package-level singleton used by BroadcastToUser.
var defaultHub *Hub

// InitHub creates the hub singleton and returns it.
func InitHub() *Hub {
	h := &Hub{
		clients: make(map[uint64]*Client),
	}
	defaultHub = h
	return h
}

// Register adds a client to the hub. If the same user already has an
// existing connection, that old connection is closed first.
func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if old, ok := h.clients[c.userID]; ok {
		// Close old connection gracefully.
		old.Close()
	}

	h.clients[c.userID] = c
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cur, ok := h.clients[c.userID]; ok && cur == c {
		delete(h.clients, c.userID)
	}
}

// BroadcastToUser sends a typed message to a single user.
// It is a package-level convenience that uses the default hub.
// The send is non-blocking: if the client's buffer is full the
// message is dropped and the connection is cleaned up.
func BroadcastToUser(userID uint64, msgType string, payload interface{}) {
	if defaultHub == nil {
		return
	}

	env := Envelope{Type: msgType, Payload: payload}
	data, err := json.Marshal(env)
	if err != nil {
		log.Printf("ws: marshal error: %v", err)
		return
	}

	defaultHub.mu.RLock()
	c, ok := defaultHub.clients[userID]
	if ok {
		select {
		case c.send <- data:
		default:
			// Buffer full – drop message and clean up.
			log.Printf("ws: send buffer full for user %d, dropping", userID)
			defaultHub.mu.RUnlock()
			defaultHub.Unregister(c)
			c.Close()
			return
		}
	}
	defaultHub.mu.RUnlock()
}
