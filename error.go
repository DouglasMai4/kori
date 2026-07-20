package kori

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
)

// HTTPError is a structured HTTP error with status code and optional details.
// Return it from a Handler to let kori write the error response.
type HTTPError struct {
	Status  int    `json:"-"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (e *HTTPError) Error() string { return e.Message }

// NewError creates an HTTPError with the given status and message.
// An optional details value is included in the JSON response.
func NewError(status int, message string, details ...any) *HTTPError {
	e := &HTTPError{Status: status, Message: message}

	if len(details) > 0 {
		e.Details = details[0]
	}

	return e
}

// BadRequest returns a 400 HTTPError.
func BadRequest(msg string, details ...any) *HTTPError {
	return NewError(http.StatusBadRequest, msg, details...)
}

// Unauthorized returns a 401 HTTPError.
func Unauthorized(msg string, details ...any) *HTTPError {
	return NewError(http.StatusUnauthorized, msg, details...)
}

// Forbidden returns a 403 HTTPError.
func Forbidden(msg string, details ...any) *HTTPError {
	return NewError(http.StatusForbidden, msg, details...)
}

// NotFound returns a 404 HTTPError.
func NotFound(msg string, details ...any) *HTTPError {
	return NewError(http.StatusNotFound, msg, details...)
}

// Conflict returns a 409 HTTPError.
func Conflict(msg string, details ...any) *HTTPError {
	return NewError(http.StatusConflict, msg, details...)
}

// UnprocessableEntity returns a 422 HTTPError.
func UnprocessableEntity(msg string, details ...any) *HTTPError {
	return NewError(http.StatusUnprocessableEntity, msg, details...)
}

// InternalServerError returns a 500 HTTPError.
func InternalServerError(msg string, details ...any) *HTTPError {
	return NewError(http.StatusInternalServerError, msg, details...)
}

// ErrorHandler is a function that writes an error response.
// Replace the default with SetErrorHandler to customize error handling.
type ErrorHandler func(w http.ResponseWriter, r *http.Request, err error)

var (
	errMu      sync.RWMutex
	errHandler ErrorHandler = defaultErrorHandler
)

// SetErrorHandler replaces the default error handler.
// The handler receives every error returned from a Handler.
//
//	SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
//	    w.Write([]byte("custom error: " + err.Error()))
//	})
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

// GetErrorHandler returns the current error handler.
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
