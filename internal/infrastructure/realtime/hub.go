// Package realtime, kullanıcıya özel canlı olay dağıtımı sağlar (API / mobil / panel).
package realtime

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
)

// Event, istemciye JSON olarak iletilen canlı olaydır.
type Event struct {
	Type        string `json:"type"`
	UnreadCount int64  `json:"unread_count"`
}

const EventInboxUpdated = "inbox.updated"

// UnreadCounter, bir kullanıcının okunmamış in-app bildirim sayısını döner.
type UnreadCounter func(ctx context.Context, userID string) (int64, error)

// Client, tek bir WebSocket aboneliğidir.
type Client struct {
	send chan []byte
}

// Hub, userID → bağlantı kümesi indeksi tutar ve olayları yalnızca ilgili
// kullanıcının bağlantılarına iletir (GoUI broadcast değil).
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}
	unread  UnreadCounter
}

// NewHub, hub'ı kurar. unread nil ise inbox olaylarında sayı 0 gider.
func NewHub(unread UnreadCounter) *Hub {
	return &Hub{
		clients: make(map[string]map[*Client]struct{}),
		unread:  unread,
	}
}

// Subscribe, kullanıcının canlı kanalına bir istemci ekler.
func (h *Hub) Subscribe(userID string) *Client {
	userID = strings.TrimSpace(userID)
	c := &Client{send: make(chan []byte, 16)}
	if userID == "" {
		return c
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[userID] == nil {
		h.clients[userID] = make(map[*Client]struct{})
	}
	h.clients[userID][c] = struct{}{}
	return c
}

// Unsubscribe, istemciyi çıkarır ve gönderim kanalını kapatır.
func (h *Hub) Unsubscribe(userID string, c *Client) {
	if c == nil {
		return
	}
	userID = strings.TrimSpace(userID)
	h.mu.Lock()
	if m := h.clients[userID]; m != nil {
		if _, ok := m[c]; ok {
			delete(m, c)
			if len(m) == 0 {
				delete(h.clients, userID)
			}
		}
	}
	h.mu.Unlock()
	close(c.send)
}

// Outbound, istemciye giden ham JSON kuyruğudur (yazma goroutine'i range eder).
func (c *Client) Outbound() <-chan []byte {
	if c == nil {
		ch := make(chan []byte)
		close(ch)
		return ch
	}
	return c.send
}

// Publish, olayı yalnızca userID'nin bağlı istemcilerine iletir.
func (h *Hub) Publish(userID string, ev Event) {
	userID = strings.TrimSpace(userID)
	if userID == "" || h == nil {
		return
	}
	data, err := json.Marshal(ev)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients[userID] {
		select {
		case c.send <- data:
		default:
			// Yavaş istemci: mesajı düşür (bağlantı kapanınca temizlenir).
		}
	}
}

// NotifyInbox, InboxRealtime sözleşmesini uygular: okunmamış sayıyı hesaplayıp
// inbox.updated olayını kullanıcının tüm canlı bağlantılarına gönderir.
func (h *Hub) NotifyInbox(userID string) {
	userID = strings.TrimSpace(userID)
	if userID == "" || h == nil {
		return
	}
	var count int64
	if h.unread != nil {
		if n, err := h.unread(context.Background(), userID); err == nil {
			count = n
		}
	}
	h.Publish(userID, Event{Type: EventInboxUpdated, UnreadCount: count})
}

// Close, tüm abonelikleri kapatır (shutdown).
func (h *Hub) Close() {
	if h == nil {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for userID, m := range h.clients {
		for c := range m {
			close(c.send)
		}
		delete(h.clients, userID)
	}
}
