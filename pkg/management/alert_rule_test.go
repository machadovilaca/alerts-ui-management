package management_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/management"
)

var _ = Describe("Alert Rules", func() {
	var (
		ctx           context.Context
		mockK8sClient *MockK8sClient
		client        management.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockK8sClient = NewMockK8sClient()
		client = management.NewClient(ctx, mockK8sClient)
	})

	Describe("CreateUserDefinedAlertRule", func() {
		var (
			alertRule monitoringv1.Rule
			options   management.Options
		)

		BeforeEach(func() {
			By("Creating a sample alert rule")
			alertRule = monitoringv1.Rule{
				Alert: "TestAlert",
				Expr:  intstr.FromString("up == 0"),
				Labels: map[string]string{
					"severity": "critical",
				},
				Annotations: map[string]string{
					"description": "Test alert description",
				},
			}

			options = management.Options{
				PrometheusRuleName:      "test-rule",
				PrometheusRuleNamespace: "test-namespace",
				GroupName:               "test-group",
			}
		})

		It("should successfully create a user defined alert rule", func() {
			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that AddRule was called with correct parameters")
			Expect(mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls).To(HaveLen(1))
			addRuleCall := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]

			By("Verifying the namespaced name")
			expectedNN := types.NamespacedName{
				Name:      "test-rule",
				Namespace: "test-namespace",
			}
			Expect(addRuleCall.NamespacedName).To(Equal(expectedNN))

			By("Verifying the group name")
			Expect(addRuleCall.GroupName).To(Equal("test-group"))

			By("Verifying the rule has the ID annotation added")
			Expect(addRuleCall.Rule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
			Expect(addRuleCall.Rule.Annotations[management.AlertRuleIdLabelKey]).NotTo(BeEmpty())

			By("Verifying other rule properties are preserved")
			Expect(addRuleCall.Rule.Alert).To(Equal("TestAlert"))
			Expect(addRuleCall.Rule.Expr.String()).To(Equal("up == 0"))
			Expect(addRuleCall.Rule.Labels).To(Equal(map[string]string{"severity": "critical"}))
			Expect(addRuleCall.Rule.Annotations["description"]).To(Equal("Test alert description"))
		})

		It("should use default group name when not specified", func() {
			options.GroupName = ""

			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that the default group name is used")
			addRuleCall := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]
			Expect(addRuleCall.GroupName).To(Equal(management.DefaultGroupName))
		})

		It("should add ID annotation to rule with nil annotations", func() {
			alertRule.Annotations = nil

			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that annotations map was created and ID was added")
			addRuleCall := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]
			Expect(addRuleCall.Rule.Annotations).NotTo(BeNil())
			Expect(addRuleCall.Rule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
		})

		It("should return error when PrometheusRuleName is empty", func() {
			options.PrometheusRuleName = ""

			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))

			By("Verifying that AddRule was not called")
			Expect(mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls).To(HaveLen(0))
		})

		It("should return error when PrometheusRuleNamespace is empty", func() {
			options.PrometheusRuleNamespace = ""

			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))

			By("Verifying that AddRule was not called")
			Expect(mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls).To(HaveLen(0))
		})

		It("should propagate k8s client errors", func() {
			expectedError := fmt.Errorf("k8s client error")
			mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleFunc = func(ctx context.Context, nn types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
				return expectedError
			}

			err := client.CreateUserDefinedAlertRule(ctx, alertRule, options)
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(expectedError))
		})

		It("should handle recording rules", func() {
			By("Creating a recording rule instead of alert rule")
			recordingRule := monitoringv1.Rule{
				Record: "test:recording:rule",
				Expr:   intstr.FromString("sum(up)"),
				Labels: map[string]string{
					"job": "test",
				},
			}

			err := client.CreateUserDefinedAlertRule(ctx, recordingRule, options)
			Expect(err).NotTo(HaveOccurred())

			By("Verifying that AddRule was called with the recording rule")
			addRuleCall := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]
			Expect(addRuleCall.Rule.Record).To(Equal("test:recording:rule"))
			Expect(addRuleCall.Rule.Alert).To(BeEmpty())
			Expect(addRuleCall.Rule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
		})

		It("should generate consistent IDs for identical rules", func() {
			By("Creating two identical rules")
			rule1 := monitoringv1.Rule{
				Alert: "TestAlert",
				Expr:  intstr.FromString("up == 0"),
				Labels: map[string]string{
					"severity": "critical",
				},
			}
			rule2 := rule1

			err1 := client.CreateUserDefinedAlertRule(ctx, rule1, options)
			Expect(err1).NotTo(HaveOccurred())

			// Reset mock calls and create second rule
			mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls = nil
			err2 := client.CreateUserDefinedAlertRule(ctx, rule2, options)
			Expect(err2).NotTo(HaveOccurred())

			By("Verifying both rules have the same ID")
			addRuleCall1 := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]
			id1 := addRuleCall1.Rule.Annotations[management.AlertRuleIdLabelKey]

			// Since we reset the calls, we need to look at the first call again
			mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls = nil
			err3 := client.CreateUserDefinedAlertRule(ctx, rule1, options)
			Expect(err3).NotTo(HaveOccurred())

			addRuleCall2 := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).AddRuleCalls[0]
			id2 := addRuleCall2.Rule.Annotations[management.AlertRuleIdLabelKey]

			Expect(id1).To(Equal(id2))
		})
	})

	Describe("DeleteRuleById", func() {
		var (
			alertRuleId string
			namespace   string
			name        string
			mockClient  *MockClient
		)

		BeforeEach(func() {
			alertRuleId = "test-alert-rule-id"
			namespace = "test-namespace"
			name = "test-rule"

			By("Setting up mock idMapper data")
			mockK8sClient.SetupMockIdMapper(alertRuleId, namespace, name)

			By("Using MockClient for DeleteRuleById tests to properly mock the idMapper")
			mockClient = NewMockClient(mockK8sClient)
		})

		Context("when rule exists and PrometheusRule has multiple groups", func() {
			BeforeEach(func() {
				By("Setting up a PrometheusRule with multiple groups and rules")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert1",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: alertRuleId,
										},
									},
									{
										Alert: "Alert2",
										Expr:  intstr.FromString("up == 1"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: "different-id",
										},
									},
								},
							},
							{
								Name: "group2",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert3",
										Expr:  intstr.FromString("up == 2"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: "another-id",
										},
									},
								},
							},
						},
					},
				}

				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}
			})

			It("should successfully delete the rule and update PrometheusRule", func() {
				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying Get was called")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.GetCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.GetCalls[0].Namespace).To(Equal(namespace))
				Expect(mockPrometheusRules.GetCalls[0].Name).To(Equal(name))

				By("Verifying Update was called with the rule removed")
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(1))
				updatedRule := mockPrometheusRules.UpdateCalls[0].Pr

				By("Verifying it should still have both groups")
				Expect(updatedRule.Spec.Groups).To(HaveLen(2))

				// Group1 should have only one rule (Alert2)
				Expect(updatedRule.Spec.Groups[0].Rules).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Rules[0].Alert).To(Equal("Alert2"))

				// Group2 should remain unchanged
				Expect(updatedRule.Spec.Groups[1].Rules).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[1].Rules[0].Alert).To(Equal("Alert3"))

				By("Verifying Delete was not called")
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})
		})

		Context("when rule exists and is the only rule in the group", func() {
			BeforeEach(func() {
				By("Setting up a PrometheusRule where the target rule is the only rule in its group")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert1",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: alertRuleId,
										},
									},
								},
							},
							{
								Name: "group2",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert2",
										Expr:  intstr.FromString("up == 1"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: "different-id",
										},
									},
								},
							},
						},
					},
				}

				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}
			})

			It("should remove the empty group and update PrometheusRule", func() {
				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying Update was called with the empty group removed")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(1))
				updatedRule := mockPrometheusRules.UpdateCalls[0].Pr

				By("Verifying it should have only one group now (group2)")
				Expect(updatedRule.Spec.Groups).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Name).To(Equal("group2"))
				Expect(updatedRule.Spec.Groups[0].Rules).To(HaveLen(1))
				Expect(updatedRule.Spec.Groups[0].Rules[0].Alert).To(Equal("Alert2"))

				By("Verifying Delete was not called")
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})
		})

		Context("when rule exists and is the only rule in the entire PrometheusRule", func() {
			BeforeEach(func() {
				By("Setting up a PrometheusRule with only the target rule")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert1",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: alertRuleId,
										},
									},
								},
							},
						},
					},
				}

				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}
			})

			It("should delete the entire PrometheusRule", func() {
				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying Delete was called instead of Update")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.DeleteCalls[0].Namespace).To(Equal(namespace))
				Expect(mockPrometheusRules.DeleteCalls[0].Name).To(Equal(name))

				By("Verifying Update was not called")
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(0))
			})
		})

		Context("when rule is matched by computed ID instead of annotation", func() {
			BeforeEach(func() {
				By("Setting up a rule without the ID annotation, relying on computed ID")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert:  "TestAlert",
										Expr:   intstr.FromString("up == 0"),
										Labels: map[string]string{"severity": "critical"},
									},
								},
							},
						},
					},
				}

				By("Using a known ID pattern for testing computed ID functionality")
				alertRuleId = "computed-rule-id-for-test"

				By("Setting up mock idMapper with the computed ID")
				mockK8sClient.SetupMockIdMapper(alertRuleId, namespace, name)

				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}
			})

			It("should successfully delete the rule using computed ID", func() {
				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying the entire PrometheusRule was deleted since it was the only rule")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(1))
			})
		})

		Context("error cases", func() {
			It("should return error when alert rule ID is not found", func() {
				nonExistentId := "non-existent-id"
				err := mockClient.DeleteRuleById(ctx, nonExistentId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("alert rule with id non-existent-id not found"))

				By("Verifying no calls were made to Get, Update, or Delete")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.GetCalls).To(HaveLen(0))
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(0))
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})

			It("should return error when PrometheusRule Get fails", func() {
				expectedError := fmt.Errorf("failed to get PrometheusRule")
				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return nil, expectedError
				}

				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err).To(Equal(expectedError))

				By("Verifying Get was called but Update and Delete were not")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.GetCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(0))
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})

			It("should return error when PrometheusRule Update fails", func() {
				By("Setting up a PrometheusRule that will trigger an Update (not Delete)")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert1",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: alertRuleId,
										},
									},
									{
										Alert: "Alert2",
										Expr:  intstr.FromString("up == 1"),
									},
								},
							},
						},
					},
				}

				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				mockPrometheusRules.GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}

				expectedError := fmt.Errorf("failed to update PrometheusRule")
				mockPrometheusRules.UpdateFunc = func(ctx context.Context, pr monitoringv1.PrometheusRule) error {
					return expectedError
				}

				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to update PrometheusRule"))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", namespace, name)))

				By("Verifying Update was called but failed")
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})

			It("should return error when PrometheusRule Delete fails", func() {
				By("Setting up a PrometheusRule that will trigger a Delete (single rule)")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "Alert1",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: alertRuleId,
										},
									},
								},
							},
						},
					},
				}

				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				mockPrometheusRules.GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}

				expectedError := fmt.Errorf("failed to delete PrometheusRule")
				mockPrometheusRules.DeleteFunc = func(ctx context.Context, namespace, name string) error {
					return expectedError
				}

				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to delete PrometheusRule"))
				Expect(err.Error()).To(ContainSubstring(fmt.Sprintf("%s/%s", namespace, name)))

				By("Verifying Delete was called but failed")
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(0))
			})
		})

		Context("when rule is not found in the PrometheusRule", func() {
			BeforeEach(func() {
				By("Setting up a PrometheusRule that doesn't contain the target rule")
				prometheusRule := &monitoringv1.PrometheusRule{
					ObjectMeta: metav1.ObjectMeta{
						Name:      name,
						Namespace: namespace,
					},
					Spec: monitoringv1.PrometheusRuleSpec{
						Groups: []monitoringv1.RuleGroup{
							{
								Name: "group1",
								Rules: []monitoringv1.Rule{
									{
										Alert: "DifferentAlert",
										Expr:  intstr.FromString("up == 0"),
										Annotations: map[string]string{
											management.AlertRuleIdLabelKey: "different-id",
										},
									},
								},
							},
						},
					},
				}

				mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface).GetFunc = func(ctx context.Context, namespace, name string) (*monitoringv1.PrometheusRule, error) {
					return prometheusRule, nil
				}
			})

			It("should succeed without making changes when rule is not found", func() {
				err := mockClient.DeleteRuleById(ctx, alertRuleId)
				Expect(err).NotTo(HaveOccurred())

				By("Verifying Get was called but Update and Delete were not")
				mockPrometheusRules := mockK8sClient.PrometheusRules().(*MockPrometheusRuleInterface)
				Expect(mockPrometheusRules.GetCalls).To(HaveLen(1))
				Expect(mockPrometheusRules.UpdateCalls).To(HaveLen(0))
				Expect(mockPrometheusRules.DeleteCalls).To(HaveLen(0))
			})
		})
	})
})
