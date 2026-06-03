package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/douglasmai4/kori"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type ChatMessage struct {
	User string `json:"user"`
	Text string `json:"text"`
	Time string `json:"time"`
}

type ChatHub struct {
	mu      sync.RWMutex
	clients map[chan ChatMessage]struct{}
}

func newChatHub() *ChatHub {
	return &ChatHub{
		clients: make(map[chan ChatMessage]struct{}),
	}
}

func (h *ChatHub) subscribe() chan ChatMessage {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan ChatMessage, 16)
	h.clients[ch] = struct{}{}
	return ch
}

func (h *ChatHub) unsubscribe(ch chan ChatMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, ch)
	close(ch)
}

func (h *ChatHub) broadcast(msg ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
		}
	}
}

func main() {
	hub := newChatHub()

	go func() {
		users := []string{"alice", "bob", "charlie"}
		texts := []string{
			"Hello, world!",
			"SSE is great for real-time updates",
			"Sending events from the server",
			"This is a chat-like example",
		}
		for i := 0; ; i++ {
			time.Sleep(3 * time.Second)
			msg := ChatMessage{
				User: users[i%len(users)],
				Text: texts[i%len(texts)],
				Time: time.Now().Format(time.RFC3339),
			}
			hub.broadcast(msg)
		}
	}()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
<title>SSE Demo</title>
<style>
body { font-family: sans-serif; margin: 2rem; }
#clock, #messages { margin: 1rem 0; }
#messages div { padding: 0.5rem; margin: 0.25rem 0; background: #f0f0f0; border-radius: 4px; }
</style>
</head>
<body>
<h1>SSE Demo</h1>
<h2>Clock</h2>
<pre id="clock">waiting...</pre>
<h2>Chat Messages</h2>
<div id="messages"></div>
<script>
const evtSource = new EventSource("/events/clock");
evtSource.onmessage = (e) => {
  document.getElementById("clock").textContent = e.data;
};

const msgSource = new EventSource("/events/chat");
msgSource.addEventListener("message", (e) => {
  const msg = JSON.parse(e.data);
  const div = document.createElement("div");
  div.textContent = "[" + msg.time + "] " + msg.user + ": " + msg.text;
  document.getElementById("messages").prepend(div);
});
msgSource.addEventListener("status", (e) => {
  const div = document.createElement("div");
  div.style.fontStyle = "italic";
  div.style.background = "#e0ffe0";
  div.textContent = e.data;
  document.getElementById("messages").prepend(div);
});
</script>
</body>
</html>`)
	})

	r.Get("/events/clock", func(w http.ResponseWriter, r *http.Request) {
		stream, err := kori.NewSSEWriter(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-r.Context().Done():
				log.Println("client disconnected from clock")
				return
			case <-ticker.C:
				if err := stream.SendData(time.Now().Format(time.RFC3339)); err != nil {
					return
				}
			}
		}
	})

	r.Get("/events/chat", func(w http.ResponseWriter, r *http.Request) {
		stream, err := kori.NewSSEWriter(w)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := stream.Send(kori.SSEEvent{
			Event: "status",
			Data:  "connected to chat",
		}); err != nil {
			return
		}

		ch := hub.subscribe()
		defer hub.unsubscribe(ch)

		pingTicker := time.NewTicker(15 * time.Second)
		defer pingTicker.Stop()

		for {
			select {
			case <-r.Context().Done():
				log.Println("client disconnected from chat")
				return
			case <-pingTicker.C:
				if err := stream.Ping(); err != nil {
					return
				}
			case msg := <-ch:
				if err := stream.SendJSON(msg); err != nil {
					return
				}
			}
		}
	})

	r.Post("/chat", func(w http.ResponseWriter, r *http.Request) {
		var msg ChatMessage
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
			return
		}
		msg.Time = time.Now().Format(time.RFC3339)
		hub.broadcast(msg)
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("SSE demo server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}
