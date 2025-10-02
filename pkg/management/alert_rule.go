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

func (c *client) GetAlertingRuleId(_ context.Context, alertRule monitoringv1.Rule) (string, error) {
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

func (c *client) CreateUserDefinedAlertRule(ctx context.Context, alertRule monitoringv1.Rule, options Options) error {
	id, err := c.GetAlertingRuleId(ctx, alertRule)
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

func (c *client) DeleteRuleById(ctx context.Context, alertRuleId string) error {
	prs, err := c.k8sClient.PrometheusRules().List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list PrometheusRules: %w", err)
	}

	for _, pr := range prs {
		updated := false
		var newGroups []monitoringv1.RuleGroup

		for _, group := range pr.Spec.Groups {
			newRules := c.filterRulesById(group.Rules, alertRuleId, &updated)

			// Only keep groups that still have rules
			if len(newRules) > 0 {
				group.Rules = newRules
				newGroups = append(newGroups, group)
			} else if len(newRules) != len(group.Rules) {
				// Group became empty due to rule deletion
				updated = true
			}
		}

		if updated {
			if len(newGroups) == 0 {
				// No groups left, delete the entire PrometheusRule
				err = c.k8sClient.PrometheusRules().Delete(ctx, pr.Namespace, pr.Name)
				if err != nil {
					return fmt.Errorf("failed to delete PrometheusRule %s/%s: %w", pr.Namespace, pr.Name, err)
				}
			} else {
				// Update PrometheusRule with remaining groups
				pr.Spec.Groups = newGroups
				err = c.k8sClient.PrometheusRules().Update(ctx, pr)
				if err != nil {
					return fmt.Errorf("failed to update PrometheusRule %s/%s: %w", pr.Namespace, pr.Name, err)
				}
			}
		}
	}

	return nil
}

func (c *client) filterRulesById(rules []monitoringv1.Rule, alertRuleId string, updated *bool) []monitoringv1.Rule {
	var newRules []monitoringv1.Rule

	for _, rule := range rules {
		if c.shouldDeleteRule(rule, alertRuleId) {
			*updated = true
			continue
		}
		newRules = append(newRules, rule)
	}

	return newRules
}

func (c *client) shouldDeleteRule(rule monitoringv1.Rule, alertRuleId string) bool {
	if rule.Annotations == nil {
		return false
	}

	id, exists := rule.Annotations[AlertRuleIdLabelKey]
	return exists && id == alertRuleId
}
