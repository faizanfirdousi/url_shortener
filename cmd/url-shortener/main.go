/*
What Is Remaining (The Next Steps)

  To build a more robust and scalable production system, consider these improvements:

  1. Improve Database Management
   * Current: The application creates the database schema on startup.
   * Next Step: Implement a database migration tool like golang-migrate/migrate to version your schema, enabling safer updates and rollbacks.

  2. Enhance Observability
   * Current: Basic logging, no application metrics.
   * Next Steps:
       * Centralized Logging: Adopt a system like Loki or Datadog to centralize and analyze logs.
       * Application Metrics: Use Prometheus to track request rates, latency, and errors for health monitoring and alerts.

  3. Modernize the Deployment Artifact
   * Current: The CI/CD pipeline copies raw files to the VM.
   * Next Step: Containerize the Go application with a Dockerfile. Your CI/CD pipeline should build and push a Docker image, which is then run on your server, making deployments more portable and
     consistent.

  4. Harden Security
   * Current: The application runs as root, and secrets are in a server file.
   * Next Steps:
       * Run as Non-Root: Configure your application to run as a non-privileged user—a critical security measure.
       * Secrets Management: Use a tool like HashiCorp Vault to manage secrets securely, avoiding plaintext storage.

  5. Plan for High Availability
   * Current: The single-server setup is a single point of failure.
   * Next Step (longer-term): Plan for redundancy with a load balancer, a PostgreSQL cluster, and a clustered Redis service.
*/

package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"url-shortener/internal/cache"
	"url-shortener/internal/config"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage/postgres"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", "123"),
	)
	log.Debug("debug messages are enabled")

	psqlInfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Postgres.Host, cfg.Postgres.Port, cfg.Postgres.User, cfg.Postgres.Password, cfg.Postgres.DBName)

	storage, err := postgres.New(psqlInfo)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}

	cache, err := cache.New(cfg.Redis.Address, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		log.Error("failed to init cache", sl.Err(err))
		os.Exit(1)
	}

	router := chi.NewRouter()

	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)

	// Health check endpoint (supports both GET and HEAD)
	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	router.Head("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// API routes
	router.Route("/url", func(r chi.Router) {
		r.Post("/", save.New(log, storage, cache))
	})

	// Serve index.html at root
	router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "frontend/index.html")
	})

	// Serve static files (CSS, JS) - must be before redirect route
	router.Get("/style.css", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/css")
		http.ServeFile(w, r, "frontend/style.css")
	})
	router.Get("/script.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(w, r, "frontend/script.js")
	})

	// Redirect route (catches all other GET requests as aliases)
	// This must be last to avoid catching static files
	router.Get("/{alias}", redirect.New(log, storage, cache))

	log.Info("starting server", slog.String("address", cfg.Address))

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:         cfg.Address,
		Handler:      router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Error("failed to start server")
		}
	}()

	log.Info("server started")

	<-done
	log.Info("stopping server")

	// TODO: move timeout to config
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("failed to stop server", sl.Err(err))
		return
	}

	// Close storage
	if err := storage.Close(); err != nil {
		log.Error("failed to close storage", sl.Err(err))
	}

	// Close cache
	if err := cache.Close(); err != nil {
		log.Error("failed to close cache", sl.Err(err))
	}

	log.Info("server stopped")
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default: // If env config is invalid, set prod settings by default due to security
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
