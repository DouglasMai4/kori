package kori

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

func serve(t *testing.T, r chi.Router, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, target, nil)
	r.ServeHTTP(rec, req)
	return rec
}

func TestVerbHelpers_RegisterCorrectMethod(t *testing.T) {
	cases := []struct {
		name   string
		reg    func(chi.Router, string, Handler, ...Option)
		method string
	}{
		{"GET", GET, http.MethodGet},
		{"POST", POST, http.MethodPost},
		{"PUT", PUT, http.MethodPut},
		{"PATCH", PATCH, http.MethodPatch},
		{"DELETE", DELETE, http.MethodDelete},
		{"HEAD", HEAD, http.MethodHead},
		{"OPTIONS", OPTIONS, http.MethodOptions},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := chi.NewRouter()
			tc.reg(r, "/thing", func(w http.ResponseWriter, r *http.Request) error {
				return Text(w, http.StatusOK, tc.method)
			})

			rec := serve(t, r, tc.method, "/thing")
			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", rec.Code)
			}
			if tc.method != http.MethodHead && rec.Body.String() != tc.method {
				t.Errorf("body = %q, want %q", rec.Body.String(), tc.method)
			}

			other := serve(t, r, http.MethodTrace, "/thing")
			if other.Code == http.StatusOK {
				t.Error("route responded to a method it was not registered for")
			}
		})
	}
}

func TestHandlerError_RoutedThroughErrorHandler(t *testing.T) {
	r := chi.NewRouter()
	GET(r, "/fail", func(w http.ResponseWriter, r *http.Request) error {
		return NotFound("nope")
	})

	rec := serve(t, r, http.MethodGet, "/fail")
	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandlerReturningNil_WritesNothingExtra(t *testing.T) {
	r := chi.NewRouter()
	GET(r, "/ok", func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusTeapot)
		return nil
	})

	rec := serve(t, r, http.MethodGet, "/ok")
	if rec.Code != http.StatusTeapot {
		t.Errorf("status = %d, want 418", rec.Code)
	}
}

func TestEmptyPattern_NormalizedToRoot(t *testing.T) {
	r := chi.NewRouter()
	GET(r, "", func(w http.ResponseWriter, r *http.Request) error {
		return Text(w, http.StatusOK, "root")
	})

	rec := serve(t, r, http.MethodGet, "/")
	if rec.Code != http.StatusOK || rec.Body.String() != "root" {
		t.Errorf("empty pattern did not map to /: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestRouteMiddleware_Applied(t *testing.T) {
	r := chi.NewRouter()

	var order []string
	mw := func(tag string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, tag)
				next.ServeHTTP(w, r)
			})
		}
	}

	GET(r, "/guarded", func(w http.ResponseWriter, r *http.Request) error {
		order = append(order, "handler")
		return NoContent(w)
	}, Use(mw("a"), mw("b")))

	serve(t, r, http.MethodGet, "/guarded")

	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "handler" {
		t.Errorf("execution order = %v, want [a b handler]", order)
	}
}

func TestRouteMiddleware_CanShortCircuit(t *testing.T) {
	r := chi.NewRouter()

	block := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		})
	}

	handlerCalled := false
	GET(r, "/secret", func(w http.ResponseWriter, r *http.Request) error {
		handlerCalled = true
		return nil
	}, Use(block))

	rec := serve(t, r, http.MethodGet, "/secret")
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
	if handlerCalled {
		t.Error("handler ran even though middleware short-circuited")
	}
}

func TestGroup_PrefixesRoutes(t *testing.T) {
	r := chi.NewRouter()
	users := Group(r, "/users")
	GET(users, "/{id}", func(w http.ResponseWriter, r *http.Request) error {
		return Text(w, http.StatusOK, chi.URLParam(r, "id"))
	})

	rec := serve(t, r, http.MethodGet, "/users/42")
	if rec.Code != http.StatusOK || rec.Body.String() != "42" {
		t.Errorf("group route failed: status=%d body=%q", rec.Code, rec.Body.String())
	}

	if serve(t, r, http.MethodGet, "/42").Code == http.StatusOK {
		t.Error("route leaked outside its group prefix")
	}
}

func TestGroup_MiddlewareAppliesToAllRoutes(t *testing.T) {
	r := chi.NewRouter()

	hits := 0
	count := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			next.ServeHTTP(w, r)
		})
	}

	api := Group(r, "/api", count)
	GET(api, "/a", func(w http.ResponseWriter, r *http.Request) error { return NoContent(w) })
	GET(api, "/b", func(w http.ResponseWriter, r *http.Request) error { return NoContent(w) })

	serve(t, r, http.MethodGet, "/api/a")
	serve(t, r, http.MethodGet, "/api/b")

	if hits != 2 {
		t.Errorf("group middleware hit count = %d, want 2", hits)
	}
}

func TestGroup_Nested(t *testing.T) {
	r := chi.NewRouter()
	api := Group(r, "/api")
	v1 := Group(api, "/v1")
	GET(v1, "/ping", func(w http.ResponseWriter, r *http.Request) error {
		return Text(w, http.StatusOK, "pong")
	})

	rec := serve(t, r, http.MethodGet, "/api/v1/ping")
	if rec.Code != http.StatusOK || rec.Body.String() != "pong" {
		t.Errorf("nested group route failed: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestGroup_RootRouteWithinPrefix(t *testing.T) {
	r := chi.NewRouter()
	users := Group(r, "/users")
	GET(users, "/", func(w http.ResponseWriter, r *http.Request) error {
		return Text(w, http.StatusOK, "list")
	})

	rec := serve(t, r, http.MethodGet, "/users")
	if rec.Code != http.StatusOK || rec.Body.String() != "list" {
		t.Errorf("group root route failed: status=%d body=%q", rec.Code, rec.Body.String())
	}
}

func TestGroup_PrefixTracking(t *testing.T) {
	r := chi.NewRouter()
	api := Group(r, "/api")
	if api.prefix != "/api" {
		t.Errorf("api.prefix = %q, want /api", api.prefix)
	}
	v1 := Group(api, "/v1")
	if v1.prefix != "/api/v1" {
		t.Errorf("v1.prefix = %q, want /api/v1", v1.prefix)
	}
}
