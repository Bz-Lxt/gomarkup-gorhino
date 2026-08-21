package ws

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"

	"gorhino/internal/shared/model"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	log   *slog.Logger
	mu    sync.Mutex
	conns map[*websocket.Conn]struct{}
	last  []byte
}

func New(log *slog.Logger) *Hub {
	return &Hub{log: log, conns: map[*websocket.Conn]struct{}{}}
}

func (h *Hub) Handle(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("ws upgrade", "err", err)
		return
	}
	h.mu.Lock()
	h.conns[c] = struct{}{}
	last := h.last
	h.mu.Unlock()
	_ = c.WriteMessage(websocket.TextMessage, []byte(`{"type":"hello","data":{"note":"percentiles are HDR approximate"}}`))
	if last != nil {
		_ = c.WriteMessage(websocket.TextMessage, last)
	}
	go func() {
		defer h.drop(c)
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}()
}

func (h *Hub) drop(c *websocket.Conn) {
	h.mu.Lock()
	delete(h.conns, c)
	h.mu.Unlock()
	_ = c.Close()
}

func (h *Hub) Broadcast(sn *model.Snapshot) {
	if sn == nil {
		return
	}
	payload, err := json.Marshal(map[string]any{"type": "frame", "data": sn})
	if err != nil {
		return
	}
	h.mu.Lock()
	h.last = payload
	conns := make([]*websocket.Conn, 0, len(h.conns))
	for c := range h.conns {
		conns = append(conns, c)
	}
	h.mu.Unlock()
	for _, c := range conns {
		if err := c.WriteMessage(websocket.TextMessage, payload); err != nil {
			h.drop(c)
		}
	}
}
