package kori

import (
	"encoding/json"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func RawJSON(w http.ResponseWriter, status int, data []byte) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write(data)
	return err
}

func Text(w http.ResponseWriter, status int, s string) error {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, err := w.Write([]byte(s))
	return err
}

func NoContent(w http.ResponseWriter) error {
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func Redirect(w http.ResponseWriter, r *http.Request, status int, url string) error {
	http.Redirect(w, r, url, status)
	return nil
}
