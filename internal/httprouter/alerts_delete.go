package httprouter

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
)

type BulkDeleteUserDefinedAlertRulesRequest struct {
	RuleIds []string `json:"ruleIds"`
}

type BulkDeleteUserDefinedAlertRulesResponse struct {
	DeletedIds []string          `json:"deletedIds"`
	Failed     map[string]string `json:"failed"`
}

func (hr *httpRouter) BulkDeleteUserDefinedAlertRules(w http.ResponseWriter, req *http.Request) {
	var payload BulkDeleteUserDefinedAlertRulesRequest
	if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
		errorResponse(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if len(payload.RuleIds) == 0 {
		errorResponse(w, http.StatusBadRequest, "ruleIds is required and must be non-empty")
		return
	}

	deleted := make([]string, 0, len(payload.RuleIds))
	failed := make(map[string]string)

	for _, id := range payload.RuleIds {
		id = strings.TrimSpace(id)
		if decoded, err := url.PathUnescape(id); err == nil {
			id = decoded
		}
		if id == "" {
			failed[id] = "failed to delete user alert: empty id"
			continue
		}

		if err := hr.managementClient.DeleteUserDefinedAlertRuleById(req.Context(), id); err != nil {
			msg := err.Error()
			switch {
			case strings.Contains(msg, "cannot delete"), strings.Contains(msg, "platform-managed"):
				failed[id] = "can't delete platform alert, you can disable it"
			default:
				failed[id] = "failed to delete user alert: " + msg
			}
			continue
		}
		deleted = append(deleted, id)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(BulkDeleteUserDefinedAlertRulesResponse{
		DeletedIds: deleted,
		Failed:     failed,
	})
}
