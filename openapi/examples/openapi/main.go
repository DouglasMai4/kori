package main

import (
	"errors"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/douglasmai4/kori"
	kopenapi "github.com/douglasmai4/kori/openapi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Priority string

const (
	Low    Priority = "low"
	Medium Priority = "medium"
	High   Priority = "high"
)

type Todo struct {
	ID        int       `json:"id"         doc:"Unique todo ID"      example:"1"`
	Title     string    `json:"title"       doc:"Title of the task"`
	Done      bool      `json:"done"        doc:"Whether completed"  example:"false"`
	Priority  Priority  `json:"priority"    doc:"Priority level"`
	CreatedAt time.Time `json:"created_at"  doc:"Creation timestamp"`
}

var ErrNotFound = errors.New("todo not found")

type Store struct {
	mu     sync.RWMutex
	items  map[int]*Todo
	nextID int
}

func NewStore() *Store {
	return &Store{items: make(map[int]*Todo), nextID: 1}
}

func (s *Store) List(priority Priority, done *bool) []*Todo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Todo, 0, len(s.items))
	for _, t := range s.items {
		if priority != "" && t.Priority != priority {
			continue
		}
		if done != nil && t.Done != *done {
			continue
		}
		out = append(out, t)
	}
	return out
}

func (s *Store) Get(id int) (*Todo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	return t, nil
}

func (s *Store) Create(title string, p Priority) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := &Todo{ID: s.nextID, Title: title, Priority: p, CreatedAt: time.Now()}
	s.items[s.nextID] = t
	s.nextID++
	return t
}

func (s *Store) Update(id int, title *string, done *bool, priority *Priority) (*Todo, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.items[id]
	if !ok {
		return nil, ErrNotFound
	}
	if title != nil {
		t.Title = *title
	}
	if done != nil {
		t.Done = *done
	}
	if priority != nil {
		t.Priority = *priority
	}
	return t, nil
}

func (s *Store) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; !ok {
		return ErrNotFound
	}
	delete(s.items, id)
	return nil
}

type TodoIDParams struct {
	ID int `path:"id" validate:"required,min=1" doc:"Todo ID" example:"1"`
}

type ListTodosParams struct {
	Priority string `query:"priority" validate:"omitempty,oneof=low medium high" doc:"Filter by priority"`
	Done     string `query:"done"     validate:"omitempty,oneof=true false"      doc:"Filter by completion"`
	Page     int    `query:"page"     validate:"min=0"                          doc:"Page number (0-based)" example:"0"`
	Limit    int    `query:"limit"    validate:"min=0,max=100"                  doc:"Items per page"        example:"20"`
}

type CreateTodoBody struct {
	Title    string   `json:"title"    validate:"required,min=1,max=200" doc:"Title of the task"  example:"Buy milk"`
	Priority Priority `json:"priority" validate:"required,oneof=low medium high" doc:"Priority level" example:"medium"`
}

type UpdateTodoBody struct {
	Title    *string   `json:"title"    validate:"omitempty,min=1,max=200" doc:"New title"`
	Done     *bool     `json:"done"                                        doc:"Mark as done"    example:"true"`
	Priority *Priority `json:"priority" validate:"omitempty,oneof=low medium high"`
}

func health(w http.ResponseWriter, r *http.Request) error {
	return kori.JSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func listTodos(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var p ListTodosParams
		if err := kori.BindQuery(r, &p); err != nil {
			return err
		}

		var priority Priority
		if p.Priority != "" {
			priority = Priority(p.Priority)
		}

		var done *bool
		if p.Done != "" {
			v, _ := strconv.ParseBool(p.Done)
			done = &v
		}

		todos := store.List(priority, done)
		if todos == nil {
			todos = []*Todo{}
		}

		return kori.JSON(w, http.StatusOK, map[string]any{
			"data":  todos,
			"total": len(todos),
		})
	}
}

func getTodo(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			return kori.BadRequest("id must be an integer")
		}
		todo, err := store.Get(id)
		if err != nil {
			return kori.NotFound("todo not found")
		}
		return kori.JSON(w, http.StatusOK, todo)
	}
}

func createTodo(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var body CreateTodoBody
		if err := kori.BindJSON(r, &body); err != nil {
			return err
		}
		todo := store.Create(body.Title, body.Priority)
		return kori.JSON(w, http.StatusCreated, todo)
	}
}

func updateTodo(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			return kori.BadRequest("id must be an integer")
		}
		var body UpdateTodoBody
		if err := kori.BindJSON(r, &body); err != nil {
			return err
		}
		todo, err := store.Update(id, body.Title, body.Done, body.Priority)
		if err != nil {
			return kori.NotFound("todo not found")
		}
		return kori.JSON(w, http.StatusOK, todo)
	}
}

func deleteTodo(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		id, err := strconv.Atoi(chi.URLParam(r, "id"))
		if err != nil {
			return kori.BadRequest("id must be an integer")
		}
		if err := store.Delete(id); err != nil {
			return kori.NotFound("todo not found")
		}
		return kori.NoContent(w)
	}
}

func apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") != "secret" {
			http.Error(w, `{"message":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	store := NewStore()
	store.Create("Buy milk", Low)
	store.Create("Write tests", High)
	store.Create("Ship Kori v0.2", High)

	spec := kopenapi.NewSpec(kopenapi.Config{
		Title:       "Todo API",
		Version:     "1.0.0",
		Description: "A simple todo API built with Kori + kori/openapi.",
		Servers: []kopenapi.Server{
			{URL: "http://localhost:8080", Description: "Local"},
		},
	})

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Logger)

	kori.GET(r, "/health", health)

	kori.GET(r, "/todos", listTodos(store),
		spec.Route(kopenapi.RouteConfig{
			Summary:     "List todos",
			Description: "Returns all todos, optionally filtered by priority and completion status.",
			Tags:        []string{"todos"},
			Params:      ListTodosParams{},
			Responses: map[int]any{
				200: []*Todo{},
				422: kori.HTTPError{},
			},
		}),
	)

	kori.GET(r, "/todos/{id}", getTodo(store),
		spec.Route(kopenapi.RouteConfig{
			Summary: "Get todo by ID",
			Tags:    []string{"todos"},
			Params:  TodoIDParams{},
			Responses: map[int]any{
				200: Todo{},
				400: kori.HTTPError{},
				404: kori.HTTPError{},
			},
		}),
	)

	kori.POST(r, "/todos", createTodo(store),
		spec.Route(kopenapi.RouteConfig{
			Summary: "Create todo",
			Tags:    []string{"todos"},
			Body:    CreateTodoBody{},
			Responses: map[int]any{
				201: Todo{},
				422: kori.HTTPError{},
			},
		}),
	)

	kori.PATCH(r, "/todos/{id}", updateTodo(store),
		spec.Route(kopenapi.RouteConfig{
			Summary: "Update todo",
			Tags:    []string{"todos"},
			Params:  TodoIDParams{},
			Body:    UpdateTodoBody{},
			Responses: map[int]any{
				200: Todo{},
				400: kori.HTTPError{},
				404: kori.HTTPError{},
				422: kori.HTTPError{},
			},
		}),
	)

	kori.DELETE(r, "/todos/{id}", deleteTodo(store),
		kori.Use(apiKeyAuth),
		spec.Route(kopenapi.RouteConfig{
			Summary:     "Delete todo",
			Description: "Requires X-Api-Key header.",
			Tags:        []string{"todos"},
			Params:      TodoIDParams{},
			Responses: map[int]any{
				204: nil,
				401: kori.HTTPError{},
				404: kori.HTTPError{},
			},
		}),
	)

	r.Get("/openapi.json", spec.JSONHandler())
	r.Get("/openapi.yaml", spec.YAMLHandler())
	r.Get("/docs", spec.ScalarHandler("/openapi.json"))

	log.Info("server started",
		"addr", ":8080",
		"docs", "http://localhost:8080/docs",
		"spec", "http://localhost:8080/openapi.json",
	)

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
