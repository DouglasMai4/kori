package kori

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type nonFlusher struct{ h http.Header }

func (n *nonFlusher) Header() http.Header       { return n.h }
func (n *nonFlusher) Write([]byte) (int, error) { return 0, nil }
func (n *nonFlusher) WriteHeader(int)           {}

type errWriter struct{ h http.Header }

func (e *errWriter) Header() http.Header       { return e.h }
func (e *errWriter) Write([]byte) (int, error) { return 0, errors.New("connection reset") }
func (e *errWriter) WriteHeader(int)           {}
func (e *errWriter) Flush()                    {}

func TestNewSSEWriter_SetsHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	s, err := NewSSEWriter(rec)
	if err != nil {
		t.Fatalf("NewSSEWriter: %v", err)
	}
	if s == nil {
		t.Fatal("SSEWriter is nil")
	}

	h := rec.Header()
	checks := map[string]string{
		"Content-Type":      "text/event-stream",
		"Cache-Control":     "no-cache",
		"Connection":        "keep-alive",
		"X-Accel-Buffering": "no",
	}
	for k, want := range checks {
		if got := h.Get(k); got != want {
			t.Errorf("header %s = %q, want %q", k, got, want)
		}
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestNewSSEWriter_NoFlusher(t *testing.T) {
	_, err := NewSSEWriter(&nonFlusher{h: http.Header{}})
	if !errors.Is(err, ErrStreamingNotSupported) {
		t.Errorf("err = %v, want ErrStreamingNotSupported", err)
	}
}

func TestSSE_Send_FullEvent(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	err := s.Send(SSEEvent{
		ID:    "1",
		Event: "update",
		Data:  "hello",
		Retry: 3000,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	want := "id: 1\nevent: update\nretry: 3000\ndata: hello\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("frame = %q, want %q", got, want)
	}
}

func TestSSE_Send_MultilineDataSplit(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	if err := s.SendData("line1\nline2"); err != nil {
		t.Fatal(err)
	}
	want := "data: line1\ndata: line2\n\n"
	if got := rec.Body.String(); got != want {
		t.Errorf("frame = %q, want %q", got, want)
	}
}

func TestSSE_SendData_Minimal(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	if err := s.SendData("x"); err != nil {
		t.Fatal(err)
	}
	if got := rec.Body.String(); got != "data: x\n\n" {
		t.Errorf("frame = %q, want %q", got, "data: x\n\n")
	}
}

func TestSSE_SendJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	if err := s.SendJSON(map[string]int{"n": 7}); err != nil {
		t.Fatal(err)
	}
	if got := rec.Body.String(); got != "data: {\"n\":7}\n\n" {
		t.Errorf("frame = %q", got)
	}
}

func TestSSE_SendJSON_MarshalError(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	err := s.SendJSON(make(chan int))
	if err == nil {
		t.Fatal("expected marshal error, got nil")
	}
	if strings.Contains(rec.Body.String(), "data:") {
		t.Errorf("unexpected data frame written: %q", rec.Body.String())
	}
}

func TestSSE_Ping(t *testing.T) {
	rec := httptest.NewRecorder()
	s, _ := NewSSEWriter(rec)

	if err := s.Ping(); err != nil {
		t.Fatal(err)
	}
	if got := rec.Body.String(); got != ": ping\n\n" {
		t.Errorf("ping = %q, want %q", got, ": ping\n\n")
	}
}

func TestSSE_Send_WriteErrorPropagates(t *testing.T) {
	w := &errWriter{h: http.Header{}}
	s := &SSEWriter{w: w, flusher: w}

	if err := s.Send(SSEEvent{Data: "x"}); err == nil {
		t.Error("expected write error to propagate from Send")
	}
	if err := s.SendData("y"); err == nil {
		t.Error("expected write error to propagate from SendData")
	}
	if err := s.Ping(); err == nil {
		t.Error("expected write error to propagate from Ping")
	}
}
