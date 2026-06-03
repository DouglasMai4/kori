package main

import (
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/douglasmai4/kori"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title" validate:"required,min=1,max=200"`
	Completed bool      `json:"completed"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateTodoInput struct {
	Title string `json:"title" validate:"required,min=1,max=200"`
}

type UpdateTodoInput struct {
	Title     string `json:"title" validate:"omitempty,min=1,max=200"`
	Completed *bool  `json:"completed"`
}

type ListTodoQuery struct {
	Completed *bool  `query:"completed"`
	Search    string `query:"search" validate:"omitempty,max=100"`
	Page      int    `query:"page" validate:"omitempty,min=1"`
	PageSize  int    `query:"page_size" validate:"omitempty,min=1,max=100"`
}

type PathID struct {
	ID int `path:"id" validate:"required,min=1"`
}

type PaginatedResponse struct {
	Data       any `json:"data"`
	Total      int `json:"total"`
	Page       int `json:"page"`
	PageSize   int `json:"page_size"`
	TotalPages int `json:"total_pages"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

type store struct {
	mu     sync.RWMutex
	todos  map[int]*Todo
	nextID int
}

func newStore() *store {
	return &store{
		todos:  make(map[int]*Todo),
		nextID: 1,
	}
}

func (s *store) list(completed *bool, search string, page, pageSize int) ([]*Todo, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var filtered []*Todo
	for _, t := range s.todos {
		if completed != nil && t.Completed != *completed {
			continue
		}
		if search != "" && !containsIgnoreCase(t.Title, search) {
			continue
		}
		filtered = append(filtered, t)
	}

	total := len(filtered)

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	start := (page - 1) * pageSize
	if start >= total {
		return nil, total
	}

	end := min(start+pageSize, total)

	return filtered[start:end], total
}

func (s *store) get(id int) (*Todo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, ok := s.todos[id]
	return t, ok
}

func (s *store) create(title string) *Todo {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := &Todo{
		ID:        s.nextID,
		Title:     title,
		Completed: false,
		CreatedAt: time.Now(),
	}
	s.nextID++
	s.todos[t.ID] = t
	return t
}

func (s *store) update(id int, title string, completed *bool) (*Todo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, ok := s.todos[id]
	if !ok {
		return nil, false
	}

	if title != "" {
		t.Title = title
	}
	if completed != nil {
		t.Completed = *completed
	}

	return t, true
}

func (s *store) delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.todos[id]
	if ok {
		delete(s.todos, id)
	}
	return ok
}

func containsIgnoreCase(s, substr string) bool {
	slen, sublen := len(s), len(substr)
	if slen == 0 || sublen == 0 {
		return false
	}
	for i := 0; i <= slen-sublen; i++ {
		if equalFold(s[i:i+sublen], substr) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if toLower(a[i]) != toLower(b[i]) {
			return false
		}
	}
	return true
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

func main() {
	todoStore := newStore()

	todoStore.create("Learn Go")
	todoStore.create("Build a REST API")
	todoStore.create("Write tests")

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))

	r.Get("/health", healthCheck)

	kori.GET(r, "/version", versionHandler)

	api := kori.Group(r, "/api")

	kori.GET(api, "/todos", listTodos(todoStore))
	kori.POST(api, "/todos", createTodo(todoStore))
	kori.GET(api, "/todos/{id}", getTodo(todoStore))
	kori.PUT(api, "/todos/{id}", updateTodo(todoStore))
	kori.DELETE(api, "/todos/{id}", deleteTodo(todoStore))
	kori.PATCH(api, "/todos/{id}/toggle", toggleTodo(todoStore),
		kori.Use(auditMiddleware),
	)

	admin := kori.Group(api, "/admin", adminAuthMiddleware)
	kori.GET(admin, "/stats", statsHandler(todoStore))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, `{"status":"ok"}`)
}

func versionHandler(w http.ResponseWriter, r *http.Request) error {
	return kori.JSON(w, http.StatusOK, map[string]string{
		"version": "1.0.0",
	})
}

func listTodos(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var q ListTodoQuery
		if err := kori.BindQuery(r, &q); err != nil {
			return err
		}

		if q.Page < 1 {
			q.Page = 1
		}
		if q.PageSize < 1 {
			q.PageSize = 20
		}

		todos, total := s.list(q.Completed, q.Search, q.Page, q.PageSize)
		totalPages := (total + q.PageSize - 1) / q.PageSize

		return kori.JSON(w, http.StatusOK, PaginatedResponse{
			Data:       todos,
			Total:      total,
			Page:       q.Page,
			PageSize:   q.PageSize,
			TotalPages: totalPages,
		})
	}
}

func createTodo(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var input CreateTodoInput
		if err := kori.BindJSON(r, &input); err != nil {
			return err
		}

		todo := s.create(input.Title)
		return kori.JSON(w, http.StatusCreated, todo)
	}
}

func getTodo(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id PathID
		if err := kori.Bind(r, &id); err != nil {
			return err
		}

		todo, ok := s.get(id.ID)
		if !ok {
			return kori.NotFound("todo not found")
		}

		return kori.JSON(w, http.StatusOK, todo)
	}
}

func updateTodo(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id PathID
		if err := kori.Bind(r, &id); err != nil {
			return err
		}

		var input UpdateTodoInput
		if err := kori.BindJSON(r, &input); err != nil {
			return err
		}

		_, ok := s.get(id.ID)
		if !ok {
			return kori.NotFound("todo not found")
		}

		updated, ok := s.update(id.ID, input.Title, input.Completed)
		if !ok {
			return kori.InternalServerError("failed to update todo")
		}

		return kori.JSON(w, http.StatusOK, updated)
	}
}

func deleteTodo(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id PathID
		if err := kori.Bind(r, &id); err != nil {
			return err
		}

		if !s.delete(id.ID) {
			return kori.NotFound("todo not found")
		}

		return kori.NoContent(w)
	}
}

func toggleTodo(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var id PathID
		if err := kori.Bind(r, &id); err != nil {
			return err
		}

		todo, ok := s.get(id.ID)
		if !ok {
			return kori.NotFound("todo not found")
		}

		completed := !todo.Completed
		updated, ok := s.update(id.ID, "", &completed)
		if !ok {
			return kori.InternalServerError("failed to toggle todo")
		}

		return kori.JSON(w, http.StatusOK, updated)
	}
}

func statsHandler(s *store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		todos, _ := s.list(nil, "", 1, 0)

		var completed int
		for _, t := range todos {
			if t.Completed {
				completed++
			}
		}

		return kori.JSON(w, http.StatusOK, map[string]any{
			"total_todos":     len(todos),
			"completed_todos": completed,
			"pending_todos":   len(todos) - completed,
		})
	}
}

func auditMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slog.Info("audit",
			"method", r.Method,
			"path", r.URL.Path,
			"remote", r.RemoteAddr,
		)
		next.ServeHTTP(w, r)
	})
}

func adminAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if apiKey == "" || apiKey != os.Getenv("ADMIN_API_KEY") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message":"unauthorized"}`))
			return
		}
		next.ServeHTTP(w, r)
	})
}
