package redirect

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
	"github.com/go-redis/redis/v8"

	resp "url-shortener/internal/lib/api/response"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/storage"
)

// URLGetter is an interface for getting url by alias.
//
//go:generate go run github.com/vektra/mockery/v2@v2.28.2 --name=URLGetter
type URLGetter interface {
	GetURL(alias string) (string, error)
}

type URLCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
}

func New(log *slog.Logger, urlGetter URLGetter, urlCache URLCache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		const op = "handlers.url.redirect.New"

		log := log.With(
			slog.String("op", op),
			slog.String("request_id", middleware.GetReqID(r.Context())),
		)

		alias := chi.URLParam(r, "alias")
		if alias == "" {
			log.Info("alias is empty")
			render.JSON(w, r, resp.Error("invalid request"))
			return
		}

		// Skip known static file extensions (check original path)
		path := r.URL.Path
		if len(path) > 4 {
			ext := path[len(path)-4:]
			if ext == ".css" || ext == ".js" || ext == ".png" || ext == ".jpg" || ext == ".ico" {
				http.NotFound(w, r)
				return
			}
		}

		// Check cache first
		resURL, err := urlCache.Get(r.Context(), alias)
		if err == nil {
			log.Info("got url from cache", slog.String("url", resURL))
			http.Redirect(w, r, resURL, http.StatusFound)
			return
		}
		if err != redis.Nil {
			log.Error("failed to get url from cache", sl.Err(err))
		}

		// If not in cache, get from storage
		resURL, err = urlGetter.GetURL(alias)
		if errors.Is(err, storage.ErrURLNotFound) {
			log.Info("url not found", "alias", alias)
			render.JSON(w, r, resp.Error("not found"))
			return
		}
		if err != nil {
			log.Error("failed to get url", sl.Err(err))
			render.JSON(w, r, resp.Error("internal error"))
			return
		}

		log.Info("got url from storage", slog.String("url", resURL))

		// Set to cache
		if err := urlCache.Set(r.Context(), alias, resURL, 5*time.Minute); err != nil {
			log.Error("failed to set url to cache", sl.Err(err))
		}

		// redirect to found url
		http.Redirect(w, r, resURL, http.StatusFound)
	}
}
