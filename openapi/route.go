package openapi

import (
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/douglasmai4/kori"
)

type RouteConfig struct {
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
	Params      any
	Body        any
	Responses   map[int]any
}

func (s *Spec) Route(cfg RouteConfig) kori.Option {
	return func(ri *kori.RouteInfo) {
		op := s.buildOperation(ri.Method, ri.Pattern, cfg)

		s.mu.Lock()
		defer s.mu.Unlock()

		if s.paths[ri.Pattern] == nil {
			s.paths[ri.Pattern] = make(pathItem)
		}
		s.paths[ri.Pattern][strings.ToLower(ri.Method)] = op
	}
}

func (s *Spec) buildOperation(method, pattern string, cfg RouteConfig) *Operation {
	opID := cfg.OperationID
	if opID == "" {
		opID = generateOperationID(method, pattern)
	}

	op := &Operation{
		OperationID: opID,
		Summary:     cfg.Summary,
		Description: cfg.Description,
		Tags:        cfg.Tags,
		Deprecated:  cfg.Deprecated,
		Responses:   make(Responses),
	}

	if cfg.Params != nil {
		op.Parameters = extractParams(cfg.Params, s.reg)
	}

	if cfg.Body != nil && bodyAllowed(method) {
		t := reflect.TypeOf(cfg.Body)
		for t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		bodySchema := s.reg.schemaFor(t)
		op.RequestBody = &RequestBody{
			Required: true,
			Content: map[string]MediaType{
				"application/json": {Schema: bodySchema},
			},
		}
	}

	if len(cfg.Responses) == 0 {
		op.Responses["default"] = &Response{Description: "Response"}
	} else {
		for status, body := range cfg.Responses {
			key := strconv.Itoa(status)
			desc := http.StatusText(status)
			if desc == "" {
				desc = key
			}
			resp := &Response{Description: desc}

			if body != nil {
				t := reflect.TypeOf(body)
				for t.Kind() == reflect.Ptr {
					t = t.Elem()
				}
				schema := s.reg.schemaFor(t)
				resp.Content = map[string]MediaType{
					"application/json": {Schema: schema},
				}
			}

			op.Responses[key] = resp
		}
	}

	return op
}

func bodyAllowed(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "HEAD", "DELETE", "OPTIONS", "TRACE":
		return false
	}
	return true
}
