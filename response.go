package kori

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as JSON with the given status code.
func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

// RawJSON writes pre-encoded JSON bytes with the given status code.
func RawJSON(w http.ResponseWriter, status int, data []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(data)
	return err
}

// Text writes a plain text response with the given status code.
func Text(w http.ResponseWriter, status int, s string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(s))
	return err
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// Redirect sends an HTTP redirect response.
// status should be http.StatusMovedPermanently, http.StatusFound, etc.
func Redirect(w http.ResponseWriter, r *http.Request, status int, url string) error {
	http.Redirect(w, r, url, status)
	return nil
}
