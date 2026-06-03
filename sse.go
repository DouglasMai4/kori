package kori

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var ErrStreamingNotSupported = errors.New("kori: response writer does not support streaming (does not implement http.Flusher)")

type SSEEvent struct {
	ID    string
	Event string
	Data  string
	Retry int
}

type SSEWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewSSEWriter(w http.ResponseWriter) (*SSEWriter, error) {
	fluser, ok := w.(http.Flusher)
	if !ok {
		return nil, ErrStreamingNotSupported
	}

	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	w.WriteHeader(http.StatusOK)
	fluser.Flush()

	return &SSEWriter{w: w, flusher: fluser}, nil
}

// Send writes event to the stream and flushes immediately.
//
// Returns an error if the underlying write fails, which usually means
// the client has disconnected. Callers should treat a write error as a
// normal end-of-stream signal rather than an application error:
//
//	if err := stream.Send(event); err != nil {
//	    return nil // client gone, stop the loop
//	}
func (s *SSEWriter) Send(event SSEEvent) error {
	var b strings.Builder

	if event.ID != "" {
		b.WriteString("id: ")
		b.WriteString(event.ID)
		b.WriteByte('\n')
	}

	if event.Event != "" {
		b.WriteString("event: ")
		b.WriteString(event.Event)
		b.WriteByte('\n')
	}

	if event.Retry > 0 {
		b.WriteString("retry: ")
		b.WriteString(strconv.Itoa(event.Retry))
		b.WriteByte('\n')
	}

	for line := range strings.SplitSeq(event.Data, "\n") {
		b.WriteString("data: ")
		b.WriteString(line)
		b.WriteByte('\n')
	}

	b.WriteByte('\n') // blank line = end of event

	if _, err := fmt.Fprint(s.w, b.String()); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}

// SendData sends a data-only event.
// Shorthand for Send(SSEEvent{Data: data}).
//
//	stream.SendData("hello")
//	stream.SendData(`{"status":"processing"}`)
func (s *SSEWriter) SendData(data string) error {
	return s.Send(SSEEvent{Data: data})
}

// SendJSON encodes v as JSON and sends it as a data-only event.
// Ideal for structured streaming payloads (LLM tokens, progress updates, etc.).
//
//	stream.SendJSON(map[string]string{"text": token})
//
// To set an event name alongside JSON data, encode manually and use Send:
//
//	data, _ := json.Marshal(payload)
//	stream.Send(kori.SSEEvent{Event: "token", Data: string(data)})
func (s *SSEWriter) SendJSON(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return s.SendData(string(data))
}

// Ping sends an SSE comment line to keep the connection alive.
//
// Long-lived SSE connections can be silently dropped by proxies, load
// balancers, or firewalls that close idle TCP connections. Call Ping
// periodically (e.g. every 15–30 seconds) when no real events are
// being sent to prevent this.
//
//	ticker := time.NewTicker(15 * time.Second)
//	defer ticker.Stop()
//	for {
//	    select {
//	    case <-r.Context().Done():
//	        return nil
//	    case <-ticker.C:
//	        stream.Ping()
//	    case msg := <-messages:
//	        stream.SendJSON(msg)
//	    }
//	}
func (s *SSEWriter) Ping() error {
	if _, err := fmt.Fprint(s.w, ": ping\n\n"); err != nil {
		return err
	}

	s.flusher.Flush()
	return nil
}
