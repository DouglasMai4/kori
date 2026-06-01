package kori

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

type HTTPError struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *HTTPError) Error() string { return e.Message }

func NewError(status int, message string, details ...any) *HTTPError {
	e := &HTTPError{Status: status, Message: message}

	if len(details) > 0 {
		e.Details = details[0]
	}

	return e
}

func BadRequest(msg string, details ...any) *HTTPError {
	return NewError(http.StatusBadRequest, msg, details...)
}

func Unauthorized(msg string, details ...any) *HTTPError {
	return NewError(http.StatusUnauthorized, msg, details...)
}

func Forbidden(msg string, details ...any) *HTTPError {
	return NewError(http.StatusForbidden, msg, details...)
}

func NotFound(msg string, details ...any) *HTTPError {
	return NewError(http.StatusNotFound, msg, details...)
}

func Conflict(msg string, details ...any) *HTTPError {
	return NewError(http.StatusConflict, msg, details...)
}

func UnprocessableEntity(msg string, details ...any) *HTTPError {
	return NewError(http.StatusUnprocessableEntity, msg, details...)
}

func InternalServerError(msg string, details ...any) *HTTPError {
	return NewError(http.StatusInternalServerError, msg, details...)
}

type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

var (
	errMu      sync.RWMutex
	errHandler ErrorHandler = defaultErrorHandler
)

func SetErrorHandler(h ErrorHandler) {
	errMu.Lock()
	defer errMu.Unlock()
	errHandler = h
}

func getErrorHandler() ErrorHandler {
	errMu.RLock()
	defer errMu.RUnlock()
	return errHandler
}

func GetErrorHandler() ErrorHandler {
	return getErrorHandler()
}

func defaultErrorHandler(w http.ResponseWriter, r *http.Request, err error) {
	var he *HTTPError
	if !errors.As(err, &he) {
		he = &HTTPError{
			Status:  http.StatusInternalServerError,
			Message: "internal server error",
		}
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(he.Status)
	_ = json.NewEncoder(w).Encode(he)
}
