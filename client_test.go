package seatalkbot

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_client_UpdateAccessToken(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		handlerFunc func(http.ResponseWriter, *http.Request)
		checkError  require.ErrorAssertionFunc
	}{
		{
			name: "it should return error when status code is not 200",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			checkError: require.Error,
		},
		{
			name: "it should return error when response body doesn't contain app_access_token",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`"{"other_field":"some value"}"`))
			},
			checkError: require.Error,
		},
		{
			name: "it should return nil when response body contain app_access_token",
			handlerFunc: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`"{"app_access_token":"abc"}"`))
			},
			checkError: require.NoError,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(tt.handlerFunc))
			defer server.Close()

			c, err := NewClient(Config{
				HTTPClient: &http.Client{},
				Host:       server.URL,
				AppID:      "",
				AppSecret:  "",
			})

			tt.checkError(t, err)

			if err == nil {
				assert.Equal(t, "abc", c.AccessToken())
			}
		})
	}
}