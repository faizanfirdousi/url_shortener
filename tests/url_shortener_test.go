package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/gavv/httpexpect/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/cache"
	"url-shortener/internal/http-server/handlers/redirect"
	"url-shortener/internal/http-server/handlers/url/save"
	mwLogger "url-shortener/internal/http-server/middleware/logger"
	"url-shortener/internal/lib/api"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage/postgres"
)

const (
	testUser     = "test_user"
	testPassword = "test_password"
)

func TestURLShortener_HappyPath(t *testing.T) {
	srv := startTestServer(t)
	defer srv.Close()

	e := httpexpect.Default(t, srv.URL)

	e.POST("/url").
		WithJSON(save.Request{
			URL:   gofakeit.URL(),
			Alias: random.NewRandomString(10),
		}).
		WithBasicAuth(testUser, testPassword).
		Expect().
		Status(200).
		JSON().Object().
		ContainsKey("alias")
}

func TestURLShortener_SaveRedirect(t *testing.T) {
	testCases := []struct {
		name  string
		url   string
		alias string
		error string
	}{
		{
			name:  "Valid URL",
			url:   gofakeit.URL(),
			alias: gofakeit.Word() + gofakeit.Word(),
		},
		{
			name:  "Invalid URL",
			url:   "invalid_url",
			alias: gofakeit.Word(),
			error: "field URL is not a valid URL",
		},
		{
			name:  "Empty Alias",
			url:   gofakeit.URL(),
			alias: "",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			srv := startTestServer(t)
			defer srv.Close()

			e := httpexpect.Default(t, srv.URL)

			// Save
			resp := e.POST("/url").
				WithJSON(save.Request{
					URL:   tc.url,
					Alias: tc.alias,
				}).
				WithBasicAuth(testUser, testPassword).
				Expect().Status(func() int {
					if tc.error != "" {
						return http.StatusBadRequest
					}
					return http.StatusOK
				}()).
				JSON().Object()

			if tc.error != "" {
				resp.NotContainsKey("alias")
				resp.Value("error").String().IsEqual(tc.error)
				return
			}

			alias := tc.alias
			if tc.alias != "" {
				resp.Value("alias").String().IsEqual(tc.alias)
			} else {
				resp.Value("alias").String().NotEmpty()
				alias = resp.Value("alias").String().Raw()
			}

			// Redirect
			testRedirect(t, srv.URL, alias, tc.url)
		})
	}
}

func startTestServer(t *testing.T) *httptest.Server {
	t.Helper()

	psqlInfo := "host=localhost port=5432 user=postgres password=password dbname=url_shortener_test sslmode=disable"
	storage, err := postgres.New(psqlInfo)
	require.NoError(t, err)

	cache, err := cache.New("localhost:6379", "", 0)
	require.NoError(t, err)

	log := slogdiscard.NewDiscardLogger()

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(mwLogger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router) {
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			testUser: testPassword,
		}))
		r.Post("/", save.New(log, storage, cache))
	})

	router.Get("/{alias}", redirect.New(log, storage, cache))

	return httptest.NewServer(router)
}

func testRedirect(t *testing.T, serverURL, alias string, urlToRedirect string) {
	t.Helper()

	redirectURL := serverURL + "/" + alias
	redirectedToURL, err := api.GetRedirect(redirectURL)
	require.NoError(t, err)
	require.Equal(t, urlToRedirect, redirectedToURL)
}