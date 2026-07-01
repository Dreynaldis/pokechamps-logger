package main

import (
	"log"
	"net/http"

	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/database"
	"github.com/dreynaldis/pokechamps-logger/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	cfg := config.Load()

	db, err := database.Connect(cfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatalf("database migration failed: %v", err)
	}

	log.Println("database connected and migrations ran")

	h := &handler.Handler{DB: db}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS for local SvelteKit dev server
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			origin := cfg.FrontendOrigin
			if origin == "" {
				origin = "http://localhost:5173"
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/health", handler.Health)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/pokemon", h.ListPokemon)
		r.Get("/pokemon/{name}", h.GetPokemon)
	})

	addr := ":" + cfg.Port
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}
