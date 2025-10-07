package mapper

import "github.com/machadovilaca/alerts-ui-management/pkg/k8s"

// New creates a new instance of the mapper client.
func New(k8sClient k8s.Client) Client {
	return &mapper{
		k8sClient: k8sClient,
		files:     make(map[PrometheusRuleId][]PrometheusAlertRuleId),
	}
}
