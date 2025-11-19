package httprouter

import (
	"encoding/json"
	"net/http"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

type CreateAlertRuleRequest struct {
	SourceRuleId            string            `json:"sourceRuleId,omitempty"`
	Alert                   string            `json:"alert,omitempty"`
	Expr                    string            `json:"expr,omitempty"`
	For                     string            `json:"for,omitempty"`
	Labels                  map[string]string `json:"labels,omitempty"`
	Annotations             map[string]string `json:"annotations,omitempty"`
	PrometheusRuleName      string            `json:"prometheusRuleName"`
	PrometheusRuleNamespace string            `json:"prometheusRuleNamespace"`
	GroupName               string            `json:"groupName,omitempty"`
}

type CreateAlertRuleResponse struct {
	Id string `json:"id"`
}

func (hr *httpRouter) CreateUserDefinedAlertRule(w http.ResponseWriter, req *http.Request) {
	var reqBody CreateAlertRuleRequest
	if err := json.NewDecoder(req.Body).Decode(&reqBody); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var rule monitoringv1.Rule

	if strings.TrimSpace(reqBody.SourceRuleId) != "" {
		existing, err := hr.managementClient.GetRuleById(req.Context(), reqBody.SourceRuleId)
		if err != nil {
			handleError(w, err)
			return
		}
		rule = existing
		rule.Alert = strings.TrimSpace(reqBody.Alert)

	} else {
		rule = monitoringv1.Rule{
			Alert:       reqBody.Alert,
			Expr:        intstr.FromString(reqBody.Expr),
			For:         monitoringv1.DurationPointer(reqBody.For),
			Labels:      reqBody.Labels,
			Annotations: reqBody.Annotations,
		}
	}

	prOpts := management.PrometheusRuleOptions{
		Name:      reqBody.PrometheusRuleName,
		Namespace: reqBody.PrometheusRuleNamespace,
		GroupName: reqBody.GroupName,
	}

	alertRuleId, err := hr.managementClient.CreateUserDefinedAlertRule(req.Context(), rule, prOpts)
	if err != nil {
		handleError(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(CreateAlertRuleResponse{Id: alertRuleId})
}
