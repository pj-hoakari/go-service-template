package httpapi_test

import (
	"net/http"
	"testing"

	"github.com/pj-hoakari/go-service-template/internal/infra/httpapi"
)

func TestHealthz(t *testing.T) {
	t.Parallel()

	status, body := request(t, httpapi.NewHandler(httpapi.HealthRoutes()), http.MethodGet, "/healthz")

	if want := http.StatusOK; status != want {
		t.Fatalf("status = %d, want %d", status, want)
	}

	if want := "ok"; body != want {
		t.Errorf("body = %q, want %q", body, want)
	}
}

func TestHealthzRejectsNonGet(t *testing.T) {
	t.Parallel()

	status, _ := request(t, httpapi.NewHandler(httpapi.HealthRoutes()), http.MethodPost, "/healthz")

	if want := http.StatusMethodNotAllowed; status != want {
		t.Fatalf("status = %d, want %d", status, want)
	}
}
