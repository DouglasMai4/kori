package kori

import (
	"net/http/httptest"
	"testing"
)

func TestJSON(t *testing.T) {
	rec := httptest.NewRecorder()

	if err := JSON(rec, 201, map[string]string{"hello": "world"}); err != nil {
		t.Fatalf("JSON returned error: %v", err)
	}

	if rec.Code != 201 {
		t.Errorf("status = %d, want 201", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q, want application/json; charset=utf-8", ct)
	}
	if got := rec.Body.String(); got != "{\"hello\":\"world\"}\n" {
		t.Errorf("body = %q, want %q", got, "{\"hello\":\"world\"}\n")
	}
}

func TestJSON_EncodeError(t *testing.T) {
	rec := httptest.NewRecorder()

	err := JSON(rec, 200, make(chan int))
	if err == nil {
		t.Fatal("expected an encoding error, got nil")
	}
}

func TestRawJSON(t *testing.T) {
	rec := httptest.NewRecorder()
	raw := []byte(`{"pre":"encoded"}`)

	if err := RawJSON(rec, 200, raw); err != nil {
		t.Fatalf("RawJSON returned error: %v", err)
	}

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}
	if got := rec.Body.String(); got != `{"pre":"encoded"}` {
		t.Errorf("body = %q, want verbatim input", got)
	}
}

func TestText(t *testing.T) {
	rec := httptest.NewRecorder()

	if err := Text(rec, 200, "plain body"); err != nil {
		t.Fatalf("Text returned error: %v", err)
	}

	if ct := rec.Header().Get("Content-Type"); ct != "text/plain; charset=utf-8" {
		t.Errorf("Content-Type = %q, want text/plain; charset=utf-8", ct)
	}
	if got := rec.Body.String(); got != "plain body" {
		t.Errorf("body = %q, want %q", got, "plain body")
	}
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()

	if err := NoContent(rec); err != nil {
		t.Fatalf("NoContent returned error: %v", err)
	}

	if rec.Code != 204 {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestRedirect(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/old", nil)

	if err := Redirect(rec, req, 302, "/new"); err != nil {
		t.Fatalf("Redirect returned error: %v", err)
	}

	if rec.Code != 302 {
		t.Errorf("status = %d, want 302", rec.Code)
	}
	if loc := rec.Header().Get("Location"); loc != "/new" {
		t.Errorf("Location = %q, want /new", loc)
	}
}
