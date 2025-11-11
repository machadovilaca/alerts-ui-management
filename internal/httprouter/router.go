package httprouter

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

type httpRouter struct {
	managementClient management.Client
}

func New(managementClient management.Client) *chi.Mux {
	httpRouter := &httpRouter{
		managementClient: managementClient,
	}

	r := chi.NewRouter()

	r.Get("/api/v1/alerting/health", httpRouter.GetHealth)
	r.Get("/api/v1/alerting/alerts", httpRouter.GetAlerts)

	return r
}

func errorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}
