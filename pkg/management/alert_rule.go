package management

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	AlertRuleIdLabelKey = "auim_id"
	DefaultGroupName    = "user-defined-rules"
)

var volatileAnnotationKeys = map[string]bool{
	AlertRuleIdLabelKey: true,
}

func (c client) CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) error {
	id, err := calculateAlertRuleId(alertRule)
	if err != nil {
		return err
	}

	if alertRule.Annotations == nil {
		alertRule.Annotations = make(map[string]string)
	}
	alertRule.Annotations[AlertRuleIdLabelKey] = id

	if options.PrometheusRuleName == "" || options.PrometheusRuleNamespace == "" {
		return fmt.Errorf("PrometheusRule Name and Namespace must be specified")
	}
	nn := types.NamespacedName{
		Name:      options.PrometheusRuleName,
		Namespace: options.PrometheusRuleNamespace,
	}

	if options.GroupName == "" {
		options.GroupName = DefaultGroupName
	}

	err = c.k8sClient.PrometheusRules().AddRule(ctx, nn, options.GroupName, alertRule)
	if err != nil {
		return err
	}

	return nil
}

func calculateAlertRuleId(alertRule monitoringv1.Rule) (string, error) {
	var kind, name string
	if alertRule.Alert != "" {
		kind = "alert"
		name = alertRule.Alert
	} else if alertRule.Record != "" {
		kind = "record"
		name = alertRule.Record
	} else {
		return "", fmt.Errorf("alert rule must have either 'alert' or 'record' field set")
	}

	expr := alertRule.Expr.String()
	forDuration := ""
	if alertRule.For != nil {
		forDuration = string(*alertRule.For)
	}

	var sortedLabels []string
	if alertRule.Labels != nil {
		for key, value := range alertRule.Labels {
			sortedLabels = append(sortedLabels, fmt.Sprintf("%s=%s", key, value))
		}
		sort.Strings(sortedLabels)
	}

	var sortedAnnotations []string
	if alertRule.Annotations != nil {
		for key, value := range alertRule.Annotations {
			if !volatileAnnotationKeys[key] {
				sortedAnnotations = append(sortedAnnotations, fmt.Sprintf("%s=%s", key, value))
			}
		}
		sort.Strings(sortedAnnotations)
	}

	// Build the hash input string
	hashInput := strings.Join([]string{
		kind,
		name,
		expr,
		forDuration,
		strings.Join(sortedLabels, ","),
		strings.Join(sortedAnnotations, ","),
	}, "\n")

	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(hashInput))
	id := fmt.Sprintf("%x", hash)

	return id, nil
}
