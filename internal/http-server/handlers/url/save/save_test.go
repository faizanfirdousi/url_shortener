package save_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"url-shortener/internal/http-server/handlers/url/save"
	"url-shortener/internal/http-server/handlers/url/save/mocks"
	"url-shortener/internal/lib/logger/handlers/slogdiscard"
	"url-shortener/internal/storage"
)

func TestSaveHandler(t *testing.T) {
	cases := []struct {
		name       string
		alias      string
		url        string
		respError  string
		mockError  error
		statusCode int
	}{
		{
			name:       "Success",
			alias:      "test_alias",
			url:        "https://google.com",
			statusCode: http.StatusOK,
		},
		{
			name:       "Empty alias",
			alias:      "",
			url:        "https://google.com",
			statusCode: http.StatusOK,
		},
		{
			name:       "Empty URL",
			url:        "",
			alias:      "some_alias",
			respError:  "field URL is a required field",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "Invalid URL",
			url:        "some invalid URL",
			alias:      "some_alias",
			respError:  "field URL is not a valid URL",
			statusCode: http.StatusBadRequest,
		},
		{
			name:       "SaveURL Error",
			alias:      "test_alias",
			url:        "https://google.com",
			respError:  "failed to add url",
			mockError:  errors.New("unexpected error"),
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "URL Exists",
			alias:      "test_alias",
			url:        "https://google.com",
			respError:  "url already exists",
			mockError:  storage.ErrURLExists,
			statusCode: http.StatusConflict,
		},
	}

	for _, tc := range cases {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			urlSaverMock := mocks.NewURLSaver(t)
			urlCacheMock := mocks.NewURLCache(t)

			if tc.respError == "" || tc.mockError != nil {
				urlSaverMock.On("SaveURL", tc.url, mock.AnythingOfType("string")).
					Return(int64(1), tc.mockError).
					Once()
			}

			if tc.mockError == nil && tc.respError == "" {
				// alias can be random, so we use mock.AnythingOfType
				urlCacheMock.On("Set", mock.Anything, mock.AnythingOfType("string"), tc.url, 5*time.Minute).
					Return(nil).Once()
			}

			handler := save.New(slogdiscard.NewDiscardLogger(), urlSaverMock, urlCacheMock)

			input := fmt.Sprintf(`{"url": "%s", "alias": "%s"}`, tc.url, tc.alias)

			req, err := http.NewRequest(http.MethodPost, "/save", bytes.NewReader([]byte(input)))
			require.NoError(t, err)

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			require.Equal(t, tc.statusCode, rr.Code)

			body := rr.Body.String()

			var resp save.Response

			require.NoError(t, json.Unmarshal([]byte(body), &resp))

			require.Equal(t, tc.respError, resp.Error)

			// TODO: add more checks
		})
	}
}
