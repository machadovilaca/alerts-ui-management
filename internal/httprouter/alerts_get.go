package httprouter

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/form/v4"
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
)

type GetAlertsQueryParams struct {
	Labels map[string]string `form:"labels"`
}

type GetAlertsResponse struct {
	Alerts []k8s.ActiveAlert `json:"alerts"`
}

func (hr *httpRouter) GetAlerts(w http.ResponseWriter, req *http.Request) {
	var params GetAlertsQueryParams

	if err := form.NewDecoder().Decode(&params, req.URL.Query()); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid query parameters: "+err.Error())
		return
	}

	alerts, err := hr.k8sClient.PrometheusAlerts().GetActiveAlerts(req.Context())
	if err != nil {
		errorResponse(w, http.StatusInternalServerError, "Failed to get alerts: "+err.Error())
		return
	}

	// Filter alerts based on labels if provided
	if len(params.Labels) > 0 {
		filteredAlerts := make([]k8s.ActiveAlert, 0)

		for _, alert := range alerts {
			if labelsMatch(&params, &alert) {
				filteredAlerts = append(filteredAlerts, alert)
			}
		}

		alerts = filteredAlerts
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(GetAlertsResponse{Alerts: alerts})
}

// labelsMatch checks if all labels in the request match the labels of the alert
func labelsMatch(req *GetAlertsQueryParams, alert *k8s.ActiveAlert) bool {
	for key, value := range req.Labels {
		if alertValue, exists := alert.Labels[key]; !exists || alertValue != value {
			return false
		}
	}

	return true
}
