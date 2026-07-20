package kori

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPError_Error(t *testing.T) {
	e := &HTTPError{Status: 400, Message: "bad input"}
	if e.Error() != "bad input" {
		t.Errorf("Error() = %q, want %q", e.Error(), "bad input")
	}
}

func TestHTTPError_ImplementsError(t *testing.T) {
	var err error = &HTTPError{Status: 500, Message: "boom"}
	var he *HTTPError
	if !errors.As(err, &he) {
		t.Fatal("errors.As failed to match *HTTPError")
	}
	if he.Status != 500 {
		t.Errorf("Status = %d, want 500", he.Status)
	}
}

func TestNewError_WithoutDetails(t *testing.T) {
	e := NewError(418, "teapot")
	if e.Status != 418 || e.Message != "teapot" {
		t.Errorf("got %+v", e)
	}
	if e.Details != nil {
		t.Errorf("Details = %v, want nil", e.Details)
	}
}

func TestNewError_WithDetails(t *testing.T) {
	e := NewError(400, "bad", "field x invalid")
	if e.Details != "field x invalid" {
		t.Errorf("Details = %v, want %q", e.Details, "field x invalid")
	}
}

func TestNewError_OnlyFirstDetailUsed(t *testing.T) {
	e := NewError(400, "bad", "first", "second")
	if e.Details != "first" {
		t.Errorf("Details = %v, want %q", e.Details, "first")
	}
}

func TestErrorConstructors_Status(t *testing.T) {
	cases := []struct {
		name string
		fn   func(string, ...any) *HTTPError
		want int
	}{
		{"BadRequest", BadRequest, http.StatusBadRequest},
		{"Unauthorized", Unauthorized, http.StatusUnauthorized},
		{"Forbidden", Forbidden, http.StatusForbidden},
		{"NotFound", NotFound, http.StatusNotFound},
		{"Conflict", Conflict, http.StatusConflict},
		{"UnprocessableEntity", UnprocessableEntity, http.StatusUnprocessableEntity},
		{"InternalServerError", InternalServerError, http.StatusInternalServerError},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := tc.fn("msg")
			if e.Status != tc.want {
				t.Errorf("Status = %d, want %d", e.Status, tc.want)
			}
			if e.Message != "msg" {
				t.Errorf("Message = %q, want msg", e.Message)
			}
		})
	}
}

func TestHTTPError_JSONShape(t *testing.T) {
	e := NewError(404, "not found")
	data, err := json.Marshal(e)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(data); got != `{"message":"not found"}` {
		t.Errorf("JSON = %s, want %s", got, `{"message":"not found"}`)
	}

	withDetails := NewError(400, "bad", map[string]string{"field": "name"})
	data, _ = json.Marshal(withDetails)
	if got := string(data); got != `{"message":"bad","details":{"field":"name"}}` {
		t.Errorf("JSON = %s", got)
	}
}

func TestDefaultErrorHandler_HTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	defaultErrorHandler(rec, req, NotFound("gone", "no such id"))

	if rec.Code != 404 {
		t.Errorf("status = %d, want 404", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type = %q", ct)
	}

	var body struct {
		Message string `json:"message"`
		Details string `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "gone" || body.Details != "no such id" {
		t.Errorf("body = %+v", body)
	}
}

func TestDefaultErrorHandler_GenericErrorBecomes500(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	defaultErrorHandler(rec, req, errors.New("database password is hunter2"))

	if rec.Code != 500 {
		t.Errorf("status = %d, want 500", rec.Code)
	}

	var body struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Message != "internal server error" {
		t.Errorf("Message = %q, want %q (must not leak internals)", body.Message, "internal server error")
	}
}

func TestDefaultErrorHandler_WrappedHTTPError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)

	wrapped := fmt.Errorf("context: %w", Conflict("duplicate"))
	defaultErrorHandler(rec, req, wrapped)

	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusConflict)
	}
}

func TestSetErrorHandler_And_Get(t *testing.T) {
	t.Cleanup(func() { SetErrorHandler(defaultErrorHandler) })

	called := false
	custom := func(w http.ResponseWriter, r *http.Request, err error) {
		called = true
		w.WriteHeader(599)
	}
	SetErrorHandler(custom)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	getErrorHandler()(rec, req, errors.New("x"))

	if !called {
		t.Error("custom error handler was not invoked")
	}
	if rec.Code != 599 {
		t.Errorf("status = %d, want 599", rec.Code)
	}

	if fmt.Sprintf("%p", GetErrorHandler()) != fmt.Sprintf("%p", getErrorHandler()) {
		t.Error("GetErrorHandler and getErrorHandler disagree")
	}
}
