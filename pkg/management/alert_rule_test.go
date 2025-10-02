package management_test

import (
	"context"
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

var _ = Describe("Management", func() {
	var (
		ctx       context.Context
		mockK8s   *management.MockK8sClient
		client    management.Client
		alertRule monitoringv1.Rule
		options   management.Options
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockK8s = management.NewMockK8sClient()
		client = management.NewClient(ctx, mockK8s)

		// Set up a basic alert rule for testing
		forDuration := "5m"
		alertRule = monitoringv1.Rule{
			Alert: "TestAlert",
			Expr:  intstr.FromString("up == 0"),
			For:   (*monitoringv1.Duration)(&forDuration),
			Labels: map[string]string{
				"severity": "critical",
				"service":  "test",
			},
			Annotations: map[string]string{
				"summary":     "Test alert summary",
				"description": "Test alert description",
			},
		}

		options = management.Options{
			PrometheusRuleName:      "test-rule",
			PrometheusRuleNamespace: "test-namespace",
			GroupName:               "test-group",
		}
	})

	Describe("NewClient", func() {
		It("should create a new client successfully", func() {
			client := management.NewClient(ctx, mockK8s)
			Expect(client).NotTo(BeNil())
		})
	})

	Describe("GetAlertingRuleId", func() {
		Context("with valid alert rule", func() {
			It("should generate a consistent ID for alert rules", func() {
				id1, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())
				Expect(id1).NotTo(BeEmpty())
				Expect(len(id1)).To(Equal(64)) // SHA256 hex string length

				// Same rule should generate same ID
				id2, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())
				Expect(id2).To(Equal(id1))
			})

			It("should generate different IDs for different alert rules", func() {
				id1, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				// Modify the alert rule
				alertRule.Alert = "DifferentAlert"
				id2, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				Expect(id1).NotTo(Equal(id2))
			})

			It("should handle rules with different labels", func() {
				id1, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				// Add a label
				alertRule.Labels["new_label"] = "new_value"
				id2, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				Expect(id1).NotTo(Equal(id2))
			})

			It("should handle rules with different annotations", func() {
				id1, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				// Add an annotation
				alertRule.Annotations["new_annotation"] = "new_value"
				id2, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				Expect(id1).NotTo(Equal(id2))
			})

			It("should ignore volatile annotations", func() {
				id1, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				// Add volatile annotation
				alertRule.Annotations[management.AlertRuleIdLabelKey] = "some-id"
				id2, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())

				Expect(id1).To(Equal(id2))
			})
		})

		Context("with valid record rule", func() {
			It("should generate ID for record rules", func() {
				recordRule := monitoringv1.Rule{
					Record: "test:record",
					Expr:   intstr.FromString("sum(up)"),
				}

				id, err := client.GetAlertingRuleId(ctx, recordRule)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())
			})
		})

		Context("with invalid rule", func() {
			It("should return error for rule without alert or record", func() {
				invalidRule := monitoringv1.Rule{
					Expr: intstr.FromString("up == 0"),
				}

				id, err := client.GetAlertingRuleId(ctx, invalidRule)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have either 'alert' or 'record' field set"))
				Expect(id).To(BeEmpty())
			})
		})

		Context("with nil labels and annotations", func() {
			It("should handle nil labels and annotations", func() {
				alertRule.Labels = nil
				alertRule.Annotations = nil

				id, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())
			})
		})

		Context("with nil For duration", func() {
			It("should handle nil For duration", func() {
				alertRule.For = nil

				id, err := client.GetAlertingRuleId(ctx, alertRule)
				Expect(err).NotTo(HaveOccurred())
				Expect(id).NotTo(BeEmpty())
			})
		})
	})

	Describe("CreateUserDefinedAlertRule", func() {
		Context("with valid options", func() {
			It("should create alert rule successfully", func() {
				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).NotTo(HaveOccurred())

				// Verify the AddRule was called
				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				Expect(mockPrometheus.AddRuleCalls).To(HaveLen(1))

				addRuleCall := mockPrometheus.AddRuleCalls[0]
				Expect(addRuleCall.Ctx).To(Equal(ctx))
				Expect(addRuleCall.NamespacedName.Name).To(Equal(options.PrometheusRuleName))
				Expect(addRuleCall.NamespacedName.Namespace).To(Equal(options.PrometheusRuleNamespace))
				Expect(addRuleCall.GroupName).To(Equal(options.GroupName))

				// Verify the rule has the ID annotation added
				Expect(addRuleCall.Rule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
				Expect(addRuleCall.Rule.Annotations[management.AlertRuleIdLabelKey]).NotTo(BeEmpty())
			})

			It("should use default group name when not specified", func() {
				options.GroupName = ""

				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).NotTo(HaveOccurred())

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				Expect(mockPrometheus.AddRuleCalls).To(HaveLen(1))
				Expect(mockPrometheus.AddRuleCalls[0].GroupName).To(Equal(management.DefaultGroupName))
			})

			It("should initialize annotations map if nil", func() {
				alertRule.Annotations = nil

				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).NotTo(HaveOccurred())

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				Expect(mockPrometheus.AddRuleCalls).To(HaveLen(1))
				Expect(mockPrometheus.AddRuleCalls[0].Rule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
			})
		})

		Context("with invalid options", func() {
			It("should return error when PrometheusRuleName is empty", func() {
				options.PrometheusRuleName = ""

				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))
			})

			It("should return error when PrometheusRuleNamespace is empty", func() {
				options.PrometheusRuleNamespace = ""

				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))
			})
		})

		Context("when GetAlertingRuleId fails", func() {
			It("should return error from GetAlertingRuleId", func() {
				invalidRule := monitoringv1.Rule{
					Expr: intstr.FromString("up == 0"),
				}

				err := client.CreateUserDefinedAlertRule(ctx, invalidRule, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("must have either 'alert' or 'record' field set"))
			})
		})

		Context("when k8s AddRule fails", func() {
			It("should return error from k8s client", func() {
				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.AddRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					return errors.New("k8s error")
				}

				err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(Equal("k8s error"))
			})
		})
	})

	Describe("DeleteRuleById", func() {
		var testRuleId string

		BeforeEach(func() {
			var err error
			testRuleId, err = client.GetAlertingRuleId(ctx, alertRule)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("when rule exists", func() {
			It("should delete rule and update PrometheusRule", func() {
				// Setup mock to return a PrometheusRule with our rule
				existingRule := alertRule
				existingRule.Annotations = map[string]string{
					management.AlertRuleIdLabelKey: testRuleId,
					"summary":                      "Test alert summary",
				}

				otherRule := monitoringv1.Rule{
					Alert: "OtherAlert",
					Expr:  intstr.FromString("up == 1"),
					Annotations: map[string]string{
						management.AlertRuleIdLabelKey: "other-id",
					},
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: []monitoringv1.Rule{existingRule, otherRule},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).NotTo(HaveOccurred())

				// Verify Update was called
				Expect(mockPrometheus.UpdateCalls).To(HaveLen(1))

				updatedRule := mockPrometheus.UpdateCalls[0].Pr
				Expect(updatedRule.Spec.Groups).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Rules).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Rules[0].Alert).To(Equal("OtherAlert"))
			})

			It("should delete entire PrometheusRule when no rules remain", func() {
				// Setup mock to return a PrometheusRule with only our rule
				existingRule := alertRule
				existingRule.Annotations = map[string]string{
					management.AlertRuleIdLabelKey: testRuleId,
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: []monitoringv1.Rule{existingRule},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).NotTo(HaveOccurred())

				// Verify Delete was called instead of Update
				Expect(mockPrometheus.DeleteCalls).To(HaveLen(1))
				Expect(mockPrometheus.UpdateCalls).To(HaveLen(0))

				deleteCall := mockPrometheus.DeleteCalls[0]
				Expect(deleteCall.Namespace).To(Equal("test-namespace"))
				Expect(deleteCall.Name).To(Equal("test-rule"))
			})

			It("should handle multiple groups correctly", func() {
				// Setup rule to delete
				ruleToDelete := alertRule
				ruleToDelete.Annotations = map[string]string{
					management.AlertRuleIdLabelKey: testRuleId,
				}

				// Setup rule to keep in different group
				ruleToKeep := monitoringv1.Rule{
					Alert: "KeepAlert",
					Expr:  intstr.FromString("up == 1"),
					Annotations: map[string]string{
						management.AlertRuleIdLabelKey: "keep-id",
					},
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "group-to-empty",
								Rules: []monitoringv1.Rule{ruleToDelete},
							},
							{
								Name:  "group-to-keep",
								Rules: []monitoringv1.Rule{ruleToKeep},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).NotTo(HaveOccurred())

				// Verify Update was called and only one group remains
				Expect(mockPrometheus.UpdateCalls).To(HaveLen(1))
				updatedRule := mockPrometheus.UpdateCalls[0].Pr
				Expect(updatedRule.Spec.Groups).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Name).To(Equal("group-to-keep"))
			})
		})

		Context("when rule does not exist", func() {
			It("should succeed when rule ID is not found", func() {
				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{}, nil
				}

				err := client.DeleteRuleById(ctx, "non-existent-id")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should handle rules without annotations", func() {
				ruleWithoutAnnotations := monitoringv1.Rule{
					Alert: "NoAnnotations",
					Expr:  intstr.FromString("up == 0"),
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: []monitoringv1.Rule{ruleWithoutAnnotations},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).NotTo(HaveOccurred())

				// Should not update since no matching rule found
				Expect(mockPrometheus.UpdateCalls).To(HaveLen(0))
				Expect(mockPrometheus.DeleteCalls).To(HaveLen(0))
			})
		})

		Context("when k8s operations fail", func() {
			It("should return error when List fails", func() {
				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return nil, errors.New("list error")
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to list PrometheusRules"))
			})

			It("should return error when Update fails", func() {
				existingRule := alertRule
				existingRule.Annotations = map[string]string{
					management.AlertRuleIdLabelKey: testRuleId,
				}

				otherRule := monitoringv1.Rule{
					Alert: "OtherAlert",
					Expr:  intstr.FromString("up == 1"),
					Annotations: map[string]string{
						management.AlertRuleIdLabelKey: "other-id",
					},
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: []monitoringv1.Rule{existingRule, otherRule},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}
				mockPrometheus.UpdateFunc = func(ctx context.Context, pr monitoringv1.PrometheusRule) error {
					return errors.New("update error")
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to update PrometheusRule"))
			})

			It("should return error when Delete fails", func() {
				existingRule := alertRule
				existingRule.Annotations = map[string]string{
					management.AlertRuleIdLabelKey: testRuleId,
				}

				prometheusRule := monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-rule",
						Namespace: "test-namespace",
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name:  "test-group",
								Rules: []monitoringv1.Rule{existingRule},
							},
						},
					},
				}

				mockPrometheus := mockK8s.PrometheusRules().(*management.MockPrometheusRuleInterface)
				mockPrometheus.ListFunc = func(ctx context.Context) ([]monitoringv1.PrometheusRule, error) {
					return []monitoringv1.PrometheusRule{prometheusRule}, nil
				}
				mockPrometheus.DeleteFunc = func(ctx context.Context, namespace string, name string) error {
					return errors.New("delete error")
				}

				err := client.DeleteRuleById(ctx, testRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to delete PrometheusRule"))
			})
		})
	})
})
