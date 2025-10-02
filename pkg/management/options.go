package management

type Options struct {
	PrometheusRuleName      string `json:"prometheusRuleName"`
	PrometheusRuleNamespace string `json:"prometheusRuleNamespace"`
	GroupName               string `json:"groupName"`
}
