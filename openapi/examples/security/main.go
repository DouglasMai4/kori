package main

import (
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/douglasmai4/kori"
	kopenapi "github.com/douglasmai4/kori/openapi"
	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
)

type Project struct {
	ID        int       `json:"id"         doc:"Project ID" example:"1"`
	Name      string    `json:"name"       doc:"Project name"`
	CreatedAt time.Time `json:"created_at" doc:"Creation timestamp"`
}

type ProjectIDParams struct {
	ID int `path:"id" validate:"required,min=1" doc:"Project ID" example:"1"`
}

type CreateProjectBody struct {
	Name string `json:"name" validate:"required,min=1,max=200" doc:"Project name" example:"My Project"`
}

type Store struct {
	mu     sync.RWMutex
	items  map[int]*Project
	nextID int
}

func NewStore() *Store {
	return &Store{items: make(map[int]*Project), nextID: 1}
}

func (s *Store) List() []*Project {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*Project, 0, len(s.items))
	for _, p := range s.items {
		out = append(out, p)
	}
	return out
}

func (s *Store) Get(id int) (*Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.items[id]
	return p, ok
}

func (s *Store) Create(name string) *Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := &Project{ID: s.nextID, Name: name, CreatedAt: time.Now()}
	s.items[s.nextID] = p
	s.nextID++
	return p
}

func (s *Store) Delete(id int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.items[id]
	if ok {
		delete(s.items, id)
	}
	return ok
}

func listProjects(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		return kori.JSON(w, http.StatusOK, store.List())
	}
}

func getProject(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var p ProjectIDParams
		if err := kori.Bind(r, &p); err != nil {
			return err
		}
		project, ok := store.Get(p.ID)
		if !ok {
			return kori.NotFound("project not found")
		}
		return kori.JSON(w, http.StatusOK, project)
	}
}

func createProject(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var body CreateProjectBody
		if err := kori.BindJSON(r, &body); err != nil {
			return err
		}
		return kori.JSON(w, http.StatusCreated, store.Create(body.Name))
	}
}

func deleteProject(store *Store) kori.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		var p ProjectIDParams
		if err := kori.Bind(r, &p); err != nil {
			return err
		}
		if !store.Delete(p.ID) {
			return kori.NotFound("project not found")
		}
		return kori.NoContent(w)
	}
}

func bearerAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, `{"message":"missing authorization header"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func apiKeyAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Api-Key") == "" {
			http.Error(w, `{"message":"missing X-Api-Key header"}`, http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))
	store := NewStore()
	store.Create("Alpha")
	store.Create("Beta")

	spec := kopenapi.NewSpec(kopenapi.Config{
		Title:       "Projects API",
		Version:     "1.0.0",
		Description: "API demonstrating OpenAPI security schemes with global and per-route overrides.",
		Servers: []kopenapi.Server{
			{URL: "http://localhost:8080", Description: "Local"},
		},
	})

	spec.AddSecurityScheme("bearer", kopenapi.BearerAuth("JWT"))
	spec.AddSecurityScheme("apiKey", kopenapi.APIKeyAuth("X-Api-Key", kopenapi.InHeader))

	spec.SetGlobalSecurity(kopenapi.Require("bearer"))

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.Logger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	api := kori.Group(r, "/api")

	kori.GET(api, "/projects", listProjects(store),
		spec.Route(kopenapi.RouteConfig{
			Summary:    "List projects",
			Tags:       []string{"projects"},
			Responses:  map[int]any{200: []*Project{}},
			NoSecurity: true,
		}),
	)

	kori.GET(api, "/projects/{id}", getProject(store),
		kori.Use(bearerAuth),
		spec.Route(kopenapi.RouteConfig{
			Summary: "Get project by ID",
			Tags:    []string{"projects"},
			Params:  ProjectIDParams{},
			Responses: map[int]any{
				200: Project{},
				404: kori.HTTPError{},
			},
		}),
	)

	kori.POST(api, "/projects", createProject(store),
		kori.Use(bearerAuth),
		spec.Route(kopenapi.RouteConfig{
			Summary: "Create project",
			Tags:    []string{"projects"},
			Body:    CreateProjectBody{},
			Responses: map[int]any{
				201: Project{},
				422: kori.HTTPError{},
			},
		}),
	)

	kori.DELETE(api, "/projects/{id}", deleteProject(store),
		kori.Use(apiKeyAuth),
		spec.Route(kopenapi.RouteConfig{
			Summary:  "Delete project",
			Tags:     []string{"projects"},
			Params:   ProjectIDParams{},
			Security: []kopenapi.SecurityRequirement{kopenapi.Require("apiKey")},
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
	)

	if err := http.ListenAndServe(":8080", r); err != nil {
		log.Error("server error", "err", err)
		os.Exit(1)
	}
}
