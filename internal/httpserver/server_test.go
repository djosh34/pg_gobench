package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"pg_gobench/internal/httpserver"
)

func TestNewServesHealthz(t *testing.T) {
	server := httpserver.New("127.0.0.1:8080")

	request := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()

	server.Handler.ServeHTTP(recorder, request)

	response := recorder.Result()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", response.StatusCode, http.StatusOK)
	}
	if got := recorder.Body.String(); got != "ok\n" {
		t.Fatalf("body = %q, want %q", got, "ok\n")
	}
}
