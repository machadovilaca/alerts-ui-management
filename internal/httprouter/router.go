package httprouter

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

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
	r.Post("/api/v1/alerting/rules", httpRouter.CreateUserDefinedAlertRule)
	r.Delete("/api/v1/alerting/rules/{ruleId}", httpRouter.DeleteUserDefinedAlertRuleById)

	return r
}

func writeError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_, _ = w.Write([]byte(`{"error":"` + message + `"}`))
}

func handleError(w http.ResponseWriter, err error) {
	var nf *management.NotFoundError
	if errors.As(err, &nf) {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	var na *management.NotAllowedError
	if errors.As(err, &na) {
		writeError(w, http.StatusMethodNotAllowed, err.Error())
		return
	}
	var ve *management.ValidationError
	if errors.As(err, &ve) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	var ce *management.ConflictError
	if errors.As(err, &ce) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	log.Printf("An unexpected error occurred: %v", err)
	writeError(w, http.StatusInternalServerError, "An unexpected error occurred")
}

func getParam(r *http.Request, name string) (string, error) {
	raw := chi.URLParam(r, name)
	decoded, err := url.PathUnescape(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s encoding", name)
	}
	value := strings.TrimSpace(decoded)
	if value == "" {
		return "", fmt.Errorf("missing %s", name)
	}
	return value, nil
}
