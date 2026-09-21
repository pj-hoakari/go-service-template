package httpapi

import (
	"log/slog"
	"net/http"
)

func HealthRoutes() Routes {
	return func(mux *http.ServeMux) {
		mux.HandleFunc("GET /healthz", handleHealthz)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		slog.ErrorContext(r.Context(), "healthz response write failed", "error", err)
	}
}
