# kori/openapi

> OpenAPI 3.1 spec generation via [`kori.Option`](https://github.com/douglasmai4/kori) — zero-reflection at runtime, full type safety at build time.

`kori/openapi` generates your OpenAPI 3.1 spec from the same Go structs you use for binding and validation. It hooks into Kori's `Option` system, so spec registration lives next to route registration — no CLI step, no YAML files, no duplication.

---

## Installation

```bash
go get github.com/douglasmai4/kori/openapi
```

Requires `github.com/douglasmai4/kori` (v0.1.0+). Go 1.22+.

---

## Quick start

```go
package main

import (
    "net/http"

    "github.com/go-chi/chi/v5"
    "github.com/douglasmai4/kori"
    kopenapi "github.com/douglasmai4/kori/openapi"
)

type Todo struct {
    ID    int    `json:"id"`
    Title string `json:"title"`
    Done  bool   `json:"done"`
}

func main() {
    spec := kopenapi.NewSpec(kopenapi.Config{
        Title:   "Todo API",
        Version: "1.0.0",
    })

    r := chi.NewRouter()

    kori.GET(r, "/todos", listTodos,
        spec.Route(kopenapi.RouteConfig{
            Summary: "List todos",
            Tags:    []string{"todos"},
            Params:  ListParams{},
            Responses: map[int]any{
                200: []Todo{},
            },
        }),
    )

    // Serve the spec
    r.Get("/openapi.json", spec.JSONHandler())
    r.Get("/docs", spec.ScalarHandler("/openapi.json"))

    http.ListenAndServe(":8080", r)
}
```

---

## Config

```go
spec := kopenapi.NewSpec(kopenapi.Config{
    Title:       "Todo API",
    Version:     "1.0.0",
    Description: "A simple todo API.",
    Servers: []kopenapi.Server{
        {URL: "http://localhost:8080", Description: "Local"},
    },
})
```

---

## Route registration

`spec.Route(cfg)` returns a `kori.Option`. Pass it as the last argument to any `kori.GET`, `kori.POST`, etc.

```go
type ListParams struct {
    Page  int    `query:"page" validate:"min=0" doc:"Page number"`
    Limit int    `query:"limit" validate:"min=1,max=100"`
    Done  string `query:"done" validate:"omitempty,oneof=true false"`
}

kori.GET(r, "/todos", listTodos,
    spec.Route(kopenapi.RouteConfig{
        OperationID: "listTodos",        // optional — auto-generated if empty
        Summary:     "List todos",
        Description: "Returns all todos, optionally filtered.",
        Tags:        []string{"todos"},
        Deprecated:  false,
        Params:      ListParams{},       // path/query/header params from struct tags
        Body:        CreateTodoBody{},   // request body schema
        Responses: map[int]any{          // status → response type (nil = no body)
            200: []Todo{},
            400: kori.HTTPError{},
            404: kori.HTTPError{},
        },
    }),
)
```

- `Params` — a struct with `path`, `query`, `header` tags. Path params are automatically `required`.
- `Body` — a struct. Skipped for GET, HEAD, DELETE, OPTIONS, TRACE.
- `Responses` — if empty, generates a `default` response with no schema.

---

## Schema generation

Structs passed to `Body` and `Responses` are automatically converted to OpenAPI schemas and registered under `#/components/schemas/`.

### Struct tags

```go
type Todo struct {
    ID        int       `json:"id"         doc:"Unique identifier"    example:"1"`
    Title     string    `json:"title"      doc:"Title of the task"   example:"Buy milk"`
    Done      bool      `json:"done"       doc:"Whether completed"   example:"false"`
    Priority  string    `json:"priority"   validate:"oneof=low medium high"`
    CreatedAt time.Time `json:"created_at"`
}
```

| Tag | Effect |
|-----|--------|
| `json` | Field name in the schema |
| `validate` | Enriches schema with `minLength`, `maxLength`, `minimum`, `maximum`, `format`, `enum` |
| `doc` | Sets `description` on the schema property |
| `example` | Sets `example` value (auto-parsed to int/bool when applicable) |

### Validation → schema mapping

| validate tag | Schema output |
|---|---|
| `required` | Added to `required` array (unless field is a pointer) |
| `min=3` (string) | `minLength: 3` |
| `max=100` (string) | `maxLength: 100` |
| `min=1` (numeric) | `minimum: 1` |
| `max=100` (numeric) | `maximum: 100` |
| `gt=0` | `exclusiveMinimum: 0` |
| `lt=100` | `exclusiveMaximum: 100` |
| `len=10` | `minLength: 10, maxLength: 10` |
| `email` | `format: email` |
| `uuid`, `uuid4` | `format: uuid` |
| `url`, `uri` | `format: uri` |
| `datetime` | `format: date-time` |
| `oneof=a b c` | `enum: ["a","b","c"]` |

Pointer fields are treated as nullable (`["type", "null"]`) and excluded from `required`.

Embedded (anonymous) structs are inlined.

---

## Serving the spec

```go
// JSON endpoint
r.Get("/openapi.json", spec.JSONHandler())

// YAML endpoint
r.Get("/openapi.yaml", spec.YAMLHandler())
```

---

## Scalar UI

[Scalar](https://scalar.com) is an API reference UI — like Swagger UI, but faster and cleaner.

```go
r.Get("/docs", spec.ScalarHandler("/openapi.json"))

// With options
r.Get("/docs", spec.ScalarHandler("/openapi.json", kopenapi.ScalarOptions{
    Theme:              "purple",
    DarkMode:           true,
    HideModels:         false,
    HideDownloadButton: false,
    DefaultOpenAllTags: true,
    CustomCSS:          "body { background: #111; }",
}))
```

---

## Auto operation IDs

If `OperationID` is empty, it is generated from the method and path pattern:

| Pattern | Method | Operation ID |
|---|---|---|
| `/todos` | GET | `list-todos` |
| `/todos/{id}` | GET | `get-todo` |
| `/todos` | POST | `create-todo` |
| `/todos/{id}` | PUT | `replace-todo` |
| `/todos/{id}` | PATCH | `update-todo` |
| `/todos/{id}` | DELETE | `delete-todo` |
| `/todos` | HEAD | `head-todos` |
| `/todos` | OPTIONS | `options-todos` |

Version segments (e.g. `v1`, `v2`) are stripped. The last noun is singularized.

---

## Coexistence with plain Chi

`kori/openapi` works with any `chi.Router`. Routes without `spec.Route(...)` are simply not documented:

```go
r.Get("/health", healthHandler)                      // undocumented
kori.GET(r, "/todos", listTodos, spec.Route(cfg))    // documented
```

---

## Full example

See [`examples/openapi/main.go`](./examples/openapi/main.go) — a TODOs API with:

- Spec config with title, version, servers
- `RouteConfig` with params, body, and responses for each endpoint
- Struct tags: `doc`, `example`, `validate`
- JSON and YAML spec endpoints
- Scalar API reference UI
- Auto-generated operation IDs

```bash
cd openapi/examples/openapi && go run .

curl http://localhost:8080/openapi.json
curl http://localhost:8080/openapi.yaml
open http://localhost:8080/docs
```

---

## License

MIT
