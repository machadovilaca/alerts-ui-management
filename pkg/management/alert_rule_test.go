package management_test

import (
	"context"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
)

var _ = Describe("AlertRule", func() {
	var (
		ctx       context.Context
		mgmClient management.Client
		mockK8s   *mockK8sClient
		alertRule monitoringv1.Rule
		options   management.Options
	)

	BeforeEach(func() {
		ctx = context.Background()
		mockK8s = &mockK8sClient{}
		mgmClient = management.NewClient(ctx, mockK8s)

		alertRule = monitoringv1.Rule{
			Alert: "TestAlert",
			Expr:  intstr.FromString("up == 0"),
			Annotations: map[string]string{
				"summary": "Test alert summary",
			},
		}

		options = management.Options{
			PrometheusRuleName:      "test-rule",
			PrometheusRuleNamespace: "test-namespace",
			GroupName:               "test-group",
		}
	})

	Describe("CreateUserDefinedAlertRule", func() {
		Context("when all required options are provided", func() {
			It("should successfully create an alert rule", func() {
				var capturedRule monitoringv1.Rule
				var capturedNamespacedName types.NamespacedName
				var capturedGroupName string

				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					capturedRule = rule
					capturedNamespacedName = namespacedName
					capturedGroupName = groupName
					return nil
				}

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).NotTo(HaveOccurred())
				Expect(capturedNamespacedName.Name).To(Equal("test-rule"))
				Expect(capturedNamespacedName.Namespace).To(Equal("test-namespace"))
				Expect(capturedGroupName).To(Equal("test-group"))
				Expect(capturedRule.Alert).To(Equal("TestAlert"))
				Expect(capturedRule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
			})

			It("should add alert rule ID annotation", func() {
				var capturedRule monitoringv1.Rule

				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					capturedRule = rule
					return nil
				}

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).NotTo(HaveOccurred())
				Expect(capturedRule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
				Expect(capturedRule.Annotations[management.AlertRuleIdLabelKey]).NotTo(BeEmpty())
			})

			It("should preserve existing annotations", func() {
				var capturedRule monitoringv1.Rule

				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					capturedRule = rule
					return nil
				}

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).NotTo(HaveOccurred())
				Expect(capturedRule.Annotations).To(HaveKey("summary"))
				Expect(capturedRule.Annotations["summary"]).To(Equal("Test alert summary"))
				Expect(capturedRule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
			})

			It("should use default group name when not specified", func() {
				var capturedGroupName string

				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					capturedGroupName = groupName
					return nil
				}

				options.GroupName = ""
				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).NotTo(HaveOccurred())
				Expect(capturedGroupName).To(Equal("user-defined-rules"))
			})
		})

		Context("when required options are missing", func() {
			It("should return error when PrometheusRuleName is empty", func() {
				options.PrometheusRuleName = ""

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))
			})

			It("should return error when PrometheusRuleNamespace is empty", func() {
				options.PrometheusRuleNamespace = ""

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))
			})
		})

		Context("when k8s client operations fail", func() {
			It("should return error when AddRule fails", func() {
				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					return fmt.Errorf("k8s error")
				}

				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("k8s error"))
			})
		})

		Context("when alert rule has no annotations", func() {
			It("should create annotations map and add ID", func() {
				var capturedRule monitoringv1.Rule

				mockK8s.prometheusRules.addRuleFunc = func(ctx context.Context, namespacedName types.NamespacedName, groupName string, rule monitoringv1.Rule) error {
					capturedRule = rule
					return nil
				}

				alertRule.Annotations = nil
				err := mgmClient.CreateUserDefinedAlertRule(ctx, alertRule, options)

				Expect(err).NotTo(HaveOccurred())
				Expect(capturedRule.Annotations).NotTo(BeNil())
				Expect(capturedRule.Annotations).To(HaveKey(management.AlertRuleIdLabelKey))
			})
		})
	})
})
