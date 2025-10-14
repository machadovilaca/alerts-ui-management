package mapper_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/testutils"
)

var _ = Describe("Mapper", func() {
	var (
		mockK8sClient *testutils.MockClient
		mapperClient  mapper.Client
	)

	BeforeEach(func() {
		mockK8sClient = &testutils.MockClient{}
		mapperClient = mapper.New(mockK8sClient)
	})

	createPrometheusRule := func(namespace, name string, alertRules []monitoringv1.Rule) *monitoringv1.PrometheusRule {
		return &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "test-group",
						Rules: alertRules,
					},
				},
			},
		}
	}

	Describe("GetAlertingRuleId", func() {
		Context("when generating IDs for alert rules", func() {
			It("should generate a non-empty ID for a simple alert rule", func() {
				By("creating a simple alert rule")
				alertRule := monitoringv1.Rule{
					Alert: "TestAlert",
					Expr:  intstr.FromString("up == 0"),
				}

				By("generating the rule ID")
				ruleId := mapperClient.GetAlertingRuleId(&alertRule)

				By("verifying the result")
				Expect(ruleId).NotTo(BeEmpty())
				Expect(string(ruleId)).To(HaveLen(64)) // SHA256 hash should be 64 characters
			})

			It("should generate different IDs for different alert rules", func() {
				By("creating two different alert rules")
				alertRule1 := monitoringv1.Rule{
					Alert: "TestAlert1",
					Expr:  intstr.FromString("up == 0"),
				}
				alertRule2 := monitoringv1.Rule{
					Alert: "TestAlert2",
					Expr:  intstr.FromString("cpu > 80"),
				}

				By("generating rule IDs")
				ruleId1 := mapperClient.GetAlertingRuleId(&alertRule1)
				ruleId2 := mapperClient.GetAlertingRuleId(&alertRule2)

				By("verifying the results")
				Expect(ruleId1).NotTo(BeEmpty())
				Expect(ruleId2).NotTo(BeEmpty())
				Expect(ruleId1).NotTo(Equal(ruleId2))
			})

			It("should generate the same ID for identical alert rules", func() {
				By("creating two identical alert rules")
				alertRule1 := monitoringv1.Rule{
					Alert: "TestAlert",
					Expr:  intstr.FromString("up == 0"),
				}
				alertRule2 := monitoringv1.Rule{
					Alert: "TestAlert",
					Expr:  intstr.FromString("up == 0"),
				}

				By("generating rule IDs")
				ruleId1 := mapperClient.GetAlertingRuleId(&alertRule1)
				ruleId2 := mapperClient.GetAlertingRuleId(&alertRule2)

				By("verifying the results")
				Expect(ruleId1).NotTo(BeEmpty())
				Expect(ruleId2).NotTo(BeEmpty())
				Expect(ruleId1).To(Equal(ruleId2))
			})

			It("should return empty string for rules without alert or record name", func() {
				By("creating a rule without alert or record name")
				alertRule := monitoringv1.Rule{
					Expr: intstr.FromString("up == 0"),
				}

				By("generating the rule ID")
				ruleId := mapperClient.GetAlertingRuleId(&alertRule)

				By("verifying the result")
				Expect(ruleId).To(BeEmpty())
			})
		})
	})

	Describe("FindAlertRuleById", func() {
		Context("when the alert rule exists", func() {
			It("should return the correct PrometheusRuleId", func() {
				By("creating test alert rule")
				alertRule := monitoringv1.Rule{
					Alert: "TestAlert",
					Expr:  intstr.FromString("up == 0"),
				}

				By("creating PrometheusRule")
				pr := createPrometheusRule("test-namespace", "test-rule", []monitoringv1.Rule{alertRule})

				By("adding the PrometheusRule to the mapper")
				mapperClient.AddPrometheusRule(pr)

				By("getting the generated rule ID")
				ruleId := mapperClient.GetAlertingRuleId(&alertRule)
				Expect(ruleId).NotTo(BeEmpty())

				By("testing FindAlertRuleById")
				foundPrometheusRuleId, _, err := mapperClient.FindAlertRuleById(ruleId)

				By("verifying results")
				Expect(err).NotTo(HaveOccurred())
				expectedPrometheusRuleId := mapper.PrometheusRuleId(types.NamespacedName{
					Namespace: "test-namespace",
					Name:      "test-rule",
				})
				Expect(*foundPrometheusRuleId).To(Equal(expectedPrometheusRuleId))
			})

			It("should return the correct PrometheusRuleId when alert rule is one of multiple in the same PrometheusRule", func() {
				By("creating multiple test alert rules")
				alertRule1 := monitoringv1.Rule{
					Alert: "TestAlert1",
					Expr:  intstr.FromString("up == 0"),
				}
				alertRule2 := monitoringv1.Rule{
					Alert: "TestAlert2",
					Expr:  intstr.FromString("cpu > 80"),
				}

				By("creating PrometheusRule with multiple rules")
				pr := createPrometheusRule("multi-namespace", "multi-rule", []monitoringv1.Rule{alertRule1, alertRule2})

				By("adding the PrometheusRule to the mapper")
				mapperClient.AddPrometheusRule(pr)

				By("getting the generated rule IDs")
				ruleId1 := mapperClient.GetAlertingRuleId(&alertRule1)
				ruleId2 := mapperClient.GetAlertingRuleId(&alertRule2)
				Expect(ruleId1).NotTo(BeEmpty())
				Expect(ruleId2).NotTo(BeEmpty())
				Expect(ruleId1).NotTo(Equal(ruleId2))

				By("testing FindAlertRuleById for both rules")
				expectedPrometheusRuleId := mapper.PrometheusRuleId(types.NamespacedName{
					Namespace: "multi-namespace",
					Name:      "multi-rule",
				})

				foundPrometheusRuleId1, _, err1 := mapperClient.FindAlertRuleById(ruleId1)
				Expect(err1).NotTo(HaveOccurred())
				Expect(*foundPrometheusRuleId1).To(Equal(expectedPrometheusRuleId))

				foundPrometheusRuleId2, _, err2 := mapperClient.FindAlertRuleById(ruleId2)
				Expect(err2).NotTo(HaveOccurred())
				Expect(*foundPrometheusRuleId2).To(Equal(expectedPrometheusRuleId))
			})
		})

		Context("when the alert rule does not exist", func() {
			It("should return an error when no rules are mapped", func() {
				By("setting up test data")
				nonExistentRuleId := mapper.PrometheusAlertRuleId("non-existent-rule-id")

				By("testing the method")
				_, _, err := mapperClient.FindAlertRuleById(nonExistentRuleId)

				By("verifying results")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("alert rule with id non-existent-rule-id not found"))
			})

			It("should return an error when rules are mapped but the target rule is not found", func() {
				By("creating and adding a valid alert rule")
				alertRule := monitoringv1.Rule{
					Alert: "ValidAlert",
					Expr:  intstr.FromString("up == 0"),
				}
				pr := createPrometheusRule("test-namespace", "test-rule", []monitoringv1.Rule{alertRule})
				mapperClient.AddPrometheusRule(pr)

				By("trying to find a non-existent rule ID")
				nonExistentRuleId := mapper.PrometheusAlertRuleId("definitely-non-existent-rule-id")

				By("testing the method")
				_, _, err := mapperClient.FindAlertRuleById(nonExistentRuleId)

				By("verifying results")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("alert rule with id definitely-non-existent-rule-id not found"))
			})
		})
	})
})
