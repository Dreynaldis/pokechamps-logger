package main

import (
	"log"
	"net/http"
	"strings"

	"github.com/dreynaldis/pokechamps-logger/internal/auth"
	"github.com/dreynaldis/pokechamps-logger/internal/config"
	"github.com/dreynaldis/pokechamps-logger/internal/database"
	"github.com/dreynaldis/pokechamps-logger/internal/handler"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/gorilla/sessions"
	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/discord"
	"github.com/markbates/goth/providers/google"
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

	setupOAuth(cfg)

	h := &handler.Handler{DB: db, Config: cfg}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS for local SvelteKit dev server
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", cfg.FrontendOrigin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			if req.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, req)
		})
	})

	r.Get("/health", handler.Health)

	// Auth routes
	r.Route("/auth", func(r chi.Router) {
		// Email/password
		r.Post("/register", h.Register)
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)
		r.Post("/logout", h.Logout)

		// OAuth -- {provider} is "google" or "discord"
		r.Get("/{provider}", h.BeginOAuth)
		r.Get("/{provider}/callback", h.OAuthCallback)
	})

	// API routes
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/pokemon", h.ListPokemon)
		r.Get("/pokemon/{name}", h.GetPokemon)

		r.Group(func(r chi.Router) {
			r.Use(auth.Middleware(cfg))
			r.Get("/auth/me", h.Me)

			// Teams
			r.Get("/teams", h.ListTeams)
			r.Post("/teams", h.CreateTeam)
			r.Get("/teams/{id}", h.GetTeam)
			r.Patch("/teams/{id}", h.PatchTeam)
			r.Delete("/teams/{id}", h.DeleteTeam)
			r.Post("/teams/{id}/activate", h.ActivateTeam)
		})
	})

	addr := ":" + cfg.Port
	log.Printf("server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, r))
}

func setupOAuth(cfg *config.Config) {
	// Secure=false for local http dev, true for https prod
	secure := !strings.HasPrefix(cfg.BaseURL, "http://localhost")

	store := sessions.NewCookieStore([]byte(cfg.AuthSecret))
	store.Options.HttpOnly = true
	store.Options.Secure = secure
	store.Options.SameSite = http.SameSiteLaxMode // Lax required: OAuth callback is a cross-site top-level redirect
	gothic.Store = store

	var providers []goth.Provider

	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		providers = append(providers, google.New(
			cfg.GoogleClientID, cfg.GoogleClientSecret,
			cfg.BaseURL+"/auth/google/callback",
			"email", "profile",
		))
		log.Println("OAuth: Google provider enabled")
	}

	if cfg.DiscordClientID != "" && cfg.DiscordClientSecret != "" {
		providers = append(providers, discord.New(
			cfg.DiscordClientID, cfg.DiscordClientSecret,
			cfg.BaseURL+"/auth/discord/callback",
			discord.ScopeIdentify, discord.ScopeEmail,
		))
		log.Println("OAuth: Discord provider enabled")
	}

	if len(providers) == 0 {
		log.Println("OAuth: no providers configured (set GOOGLE_CLIENT_ID/SECRET or DISCORD_CLIENT_ID/SECRET to enable)")
	}

	goth.UseProviders(providers...)
}
