# Kori

> A Go toolkit for [Chi](https://github.com/go-chi/chi) — inspired by [Hono](https://hono.dev), made idiomatic in Go.

Kori adds three things to vanilla Chi:

1. **Error return** — handlers return `error` instead of writing 500 by hand
2. **Binding helpers** — `BindQuery`, `BindJSON`, `BindPath`, `BindHeader`
3. **Response helpers** — `JSON`, `Text`, `NoContent`

`w http.ResponseWriter` and `r *http.Request` stay explicit. Chi is never hidden.

---

## Installation

```bash
go get github.com/douglasmai4/kori
```

Go 1.22+.

---

## Quick start

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/douglasmai4/kori"
)

func main() {
    r := chi.NewRouter()

    kori.GET(r, "/health", func(w http.ResponseWriter, r *http.Request) error {
        return kori.Text(w, http.StatusOK, "ok")
    })

    kori.GET(r, "/users/{id}", getUser)
    kori.POST(r, "/users", createUser)

    http.ListenAndServe(":8080", r)
}

func getUser(w http.ResponseWriter, r *http.Request) error {
    id := chi.URLParam(r, "id") // Chi directly, no abstraction
    user, err := db.Get(id)
    if err != nil {
        return kori.NotFound("user not found")
    }
    return kori.JSON(w, http.StatusOK, user)
}

type CreateBody struct {
    Name  string `json:"name"  validate:"required,min=2"`
    Email string `json:"email" validate:"required,email"`
}

func createUser(w http.ResponseWriter, r *http.Request) error {
    var body CreateBody
    if err := kori.BindJSON(r, &body); err != nil {
        return err // 400 or 422 automatically
    }
    user, err := db.Create(body.Name, body.Email)
    if err != nil {
        return kori.Conflict("email already in use")
    }
    return kori.JSON(w, http.StatusCreated, user)
}
```

---

## Handler

```go
type Handler func(w http.ResponseWriter, r *http.Request) error
```

It's a normal Go function with one difference: it can return `error`.

- `nil` → nothing happens (you already wrote the response)
- `*kori.HTTPError` → JSON response with the error's status code
- any other `error` → 500 Internal Server Error

`http.ResponseWriter` and `*http.Request` are always yours. You can call any stdlib function, any third-party helper, any Chi middleware — everything works as usual.

---

## Binding

### `kori.BindQuery(r, &dst)`

Populates a struct from query params using the `query` tag.

```go
type Params struct {
    Page   int      `query:"page"  validate:"min=1"`
    Limit  int      `query:"limit" validate:"min=1,max=100"`
    Search string   `query:"q"`
    Tags   []string `query:"tags"` // ?tags=a&tags=b  or  ?tags=a,b
}

func listUsers(w http.ResponseWriter, r *http.Request) error {
    var p Params
    if err := kori.BindQuery(r, &p); err != nil {
        return err
    }
    // p.Page, p.Limit, p.Search, p.Tags populated and validated
}
```

### `kori.BindPath(r, &dst)`

Populates a struct from Chi's URL params using the `path` tag.

```go
type Params struct {
    ID   string `path:"id"   validate:"required,uuid4"`
    Slug string `path:"slug" validate:"required"`
}

func getPost(w http.ResponseWriter, r *http.Request) error {
    var p Params
    if err := kori.BindPath(r, &p); err != nil {
        return err
    }
    // p.ID, p.Slug populated
}
```

> **For a single path param, prefer `chi.URLParam` directly.** `BindPath` is worth it when there are multiple params or validation.

### `kori.BindHeader(r, &dst)`

Populates a struct from headers using the `header` tag.

```go
type Params struct {
    Token   string `header:"X-Auth-Token" validate:"required"`
    Version string `header:"X-Api-Version"`
}

func handler(w http.ResponseWriter, r *http.Request) error {
    var p Params
    if err := kori.BindHeader(r, &p); err != nil {
        return err
    }
}
```

### `kori.BindJSON(r, &dst)`

Decodes the JSON body and validates the struct.

```go
type Body struct {
    Title    string `json:"title"    validate:"required,min=1,max=200"`
    Priority string `json:"priority" validate:"required,oneof=low medium high"`
    Done     *bool  `json:"done"`    // pointer = optional field
}

func createTodo(w http.ResponseWriter, r *http.Request) error {
    var body Body
    if err := kori.BindJSON(r, &body); err != nil {
        return err // 400 if invalid JSON, 422 if validation fails
    }
}
```

### `kori.Bind(r, &dst)`

Combines path, query, and header in a single call. Useful when a struct mixes sources.

```go
type Params struct {
    ID   string `path:"id"`
    Page int    `query:"page"`
    Auth string `header:"Authorization"`
}

func handler(w http.ResponseWriter, r *http.Request) error {
    var p Params
    if err := kori.Bind(r, &p); err != nil {
        return err
    }
}
```

### Supported types

`string`, `int` / `int8-64`, `uint` / `uint8-64`, `float32` / `float64`, `bool`, pointers to any of these, and slices (query only).

### Validation tags

Any [go-playground/validator](https://github.com/go-playground/validator) tag works:

```go
validate:"required"
validate:"required,min=2,max=100"
validate:"required,email"
validate:"required,uuid4"
validate:"omitempty,oneof=low medium high"
validate:"min=1,max=100"
```

Validation errors return 422 with per-field details:

```json
{
  "message": "validation failed",
  "details": [
    { "field": "email", "tag": "email", "message": "email must be a valid email address" },
    { "field": "name",  "tag": "min",   "param": "2", "message": "name must be at least 2" }
  ]
}
```

---

## Responses

All are simple functions that take `w`:

```go
kori.JSON(w, http.StatusOK, v)           // application/json
kori.JSON(w, http.StatusCreated, v)
kori.RawJSON(w, http.StatusOK, []byte{}) // pre-serialized JSON (cache, etc.)
kori.Text(w, http.StatusOK, "pong")      // text/plain
kori.NoContent(w)                        // 204, no body
kori.Redirect(w, r, http.StatusSeeOther, "/login")
```

---

## Errors

```go
// Ready-made constructors
return kori.BadRequest("invalid input")
return kori.Unauthorized("missing token")
return kori.Forbidden("insufficient permissions")
return kori.NotFound("user not found")
return kori.Conflict("email already in use")
return kori.UnprocessableEntity("validation failed")
return kori.InternalServerError("unexpected error")

// With optional details
return kori.NewError(http.StatusTooManyRequests, "rate limit exceeded", map[string]int{
    "retry_after": 60,
})

// Generic error → 500 automatically
return fmt.Errorf("db.Query: %w", err)
```

### Custom error handler

```go
kori.SetErrorHandler(func(w http.ResponseWriter, r *http.Request, err error) {
    var he *kori.HTTPError
    if !errors.As(err, &he) {
        he = kori.InternalServerError("unexpected error")
        slog.Error("unhandled", "err", err, "path", r.URL.Path)
    }

    // RFC 7807 Problem Details
    w.Header().Set("Content-Type", "application/problem+json")
    w.WriteHeader(he.Status)
    json.NewEncoder(w).Encode(map[string]any{
        "type":   "https://api.example.com/errors",
        "status": he.Status,
        "detail": he.Message,
    })
})
```

---

## Route registration

```go
kori.GET(r, "/users", listUsers)
kori.POST(r, "/users", createUser)
kori.PUT(r, "/users/{id}", replaceUser)
kori.PATCH(r, "/users/{id}", updateUser)
kori.DELETE(r, "/users/{id}", deleteUser)
kori.HEAD(r, "/users/{id}", headUser)
kori.OPTIONS(r, "/users", optionsUsers)
```

All take a `chi.Router` — any `chi.Router` works, including sub-routers.

---

## Groups

```go
// Prefix only
api := kori.Group(r, "/api/v1")
kori.GET(api, "/users", listUsers)

// Prefix + group middlewares
admin := kori.Group(r, "/admin", AuthMiddleware, AuditMiddleware)
kori.DELETE(admin, "/users/{id}", deleteUser)

// Nested
v2 := kori.Group(api, "/v2")
kori.GET(v2, "/users", listUsersV2)
```

---

## Per-route middlewares

```go
// Global (applied on the router — standard Chi)
r.Use(chimiddleware.Logger)
r.Use(chimiddleware.Recoverer)

// Per-route (applied only to this route)
kori.DELETE(r, "/admin/users/{id}", deleteUser,
    kori.Use(AuthMiddleware, AuditMiddleware),
)

// Per-group
admin := kori.Group(r, "/admin", AuthMiddleware)
```

---

## Options

`Option` is Kori's extension point. It is a function executed at route registration time, receiving `*RouteInfo` with method and pattern.

```go
type Option func(*RouteInfo)

// Internal use — kori.Use()
// External use — kori/openapi, route logging, etc.

// Example: log all routes at startup
func LogRoute(log *slog.Logger) kori.Option {
    return func(ri *kori.RouteInfo) {
        log.Info("route registered",
            "method", ri.Method,
            "pattern", ri.Pattern,
        )
    }
}

kori.GET(r, "/users", listUsers, LogRoute(log))
```

---

## Coexistence with plain Chi

Kori doesn't replace Chi — it coexists. Standard Chi handlers and Kori handlers can share the same router:

```go
r := chi.NewRouter()
r.Use(chimiddleware.Logger) // standard Chi middleware

// Standard Chi handler
r.Get("/legacy", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("legacy"))
})

// Kori handler on the same router
kori.GET(r, "/users", listUsers)
kori.POST(r, "/users", createUser)
```

---

## Full example

See [`examples/todos/main.go`](./examples/todos/main.go) — a TODOs API with:

- `BindQuery` with validation and filtering
- `chi.URLParam` directly for simple path params
- `BindJSON` with typed and validated body
- Per-route middleware (`kori.Use`)
- Group with prefix
- Implicit error handler

```bash
cd examples/todos && go run .

curl http://localhost:8080/health
curl "http://localhost:8080/api/todos?page=1&page_size=10"
curl -X POST http://localhost:8080/api/todos \
     -H "Content-Type: application/json" \
     -d '{"title":"Ship Kori"}'
curl http://localhost:8080/api/todos/1
curl -X PATCH http://localhost:8080/api/todos/1 \
     -H "Content-Type: application/json" \
     -d '{"completed":true}'
curl -X DELETE http://localhost:8080/api/todos/1
```

---

## Roadmap

- **v0.2** — `kori/openapi`: OpenAPI 3.1 spec generation via `Option` — [read more](./openapi/README.md)
- **v0.3** — `kori/testing`: test helpers for handlers, `BindMultipart` for `multipart/form-data`

---

## License

MIT
