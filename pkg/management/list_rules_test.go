package management_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/testutils"
)

var _ = Describe("ListRules", func() {
	var (
		ctx        context.Context
		mockK8s    *testutils.MockClient
		mockPR     *testutils.MockPrometheusRuleInterface
		mockMapper *testutils.MockMapperClient
		client     management.Client
	)

	BeforeEach(func() {
		ctx = context.Background()

		mockPR = &testutils.MockPrometheusRuleInterface{}
		mockK8s = &testutils.MockClient{
			PrometheusRulesFunc: func() k8s.PrometheusRuleInterface {
				return mockPR
			},
		}
		mockMapper = &testutils.MockMapperClient{}

		client = management.NewWithCustomMapper(ctx, mockK8s, mockMapper)
	})

	It("should list rules from a specific PrometheusRule", func() {
		testRule := monitoringv1.Rule{
			Alert: "TestAlert",
			Expr:  intstr.FromString("up == 0"),
		}

		prometheusRule := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-rule",
				Namespace: "test-namespace",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "test-group",
						Rules: []monitoringv1.Rule{testRule},
					},
				},
			},
		}

		mockPR.SetPrometheusRules(map[string]*monitoringv1.PrometheusRule{
			"test-namespace/test-rule": prometheusRule,
		})

		options := management.Options{
			PrometheusRuleName:      "test-rule",
			PrometheusRuleNamespace: "test-namespace",
			GroupName:               "test-group",
		}

		rules, err := client.ListRules(ctx, options)

		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(HaveLen(1))
		Expect(rules[0].Alert).To(Equal("TestAlert"))
		Expect(rules[0].Expr.String()).To(Equal("up == 0"))
	})

	It("should list rules from all namespaces", func() {
		testRule1 := monitoringv1.Rule{
			Alert: "TestAlert1",
			Expr:  intstr.FromString("up == 0"),
		}

		testRule2 := monitoringv1.Rule{
			Alert: "TestAlert2",
			Expr:  intstr.FromString("cpu_usage > 80"),
		}

		prometheusRule1 := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule1",
				Namespace: "namespace1",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "group1",
						Rules: []monitoringv1.Rule{testRule1},
					},
				},
			},
		}

		prometheusRule2 := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule2",
				Namespace: "namespace2",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "group2",
						Rules: []monitoringv1.Rule{testRule2},
					},
				},
			},
		}

		mockPR.SetPrometheusRules(map[string]*monitoringv1.PrometheusRule{
			"namespace1/rule1": prometheusRule1,
			"namespace2/rule2": prometheusRule2,
		})

		options := management.Options{}

		rules, err := client.ListRules(ctx, options)

		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(HaveLen(2))

		alertNames := []string{rules[0].Alert, rules[1].Alert}
		Expect(alertNames).To(ContainElement("TestAlert1"))
		Expect(alertNames).To(ContainElement("TestAlert2"))
	})

	It("should list all rules from a specific namespace", func() {
		// Setup test data in the same namespace but different PrometheusRules
		testRule1 := monitoringv1.Rule{
			Alert: "NamespaceAlert1",
			Expr:  intstr.FromString("memory_usage > 90"),
		}

		testRule2 := monitoringv1.Rule{
			Alert: "NamespaceAlert2",
			Expr:  intstr.FromString("disk_usage > 85"),
		}

		testRule3 := monitoringv1.Rule{
			Alert: "OtherNamespaceAlert",
			Expr:  intstr.FromString("network_error_rate > 0.1"),
		}

		// PrometheusRule in target namespace
		prometheusRule1 := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule1",
				Namespace: "target-namespace",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "group1",
						Rules: []monitoringv1.Rule{testRule1},
					},
				},
			},
		}

		// Another PrometheusRule in the same target namespace
		prometheusRule2 := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule2",
				Namespace: "target-namespace",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "group2",
						Rules: []monitoringv1.Rule{testRule2},
					},
				},
			},
		}

		// PrometheusRule in a different namespace (should not be included)
		prometheusRule3 := &monitoringv1.PrometheusRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "rule3",
				Namespace: "other-namespace",
			},
			Spec: monitoringv1.PrometheusRuleSpec{
				Groups: []monitoringv1.RuleGroup{
					{
						Name:  "group3",
						Rules: []monitoringv1.Rule{testRule3},
					},
				},
			},
		}

		mockPR.SetPrometheusRules(map[string]*monitoringv1.PrometheusRule{
			"target-namespace/rule1": prometheusRule1,
			"target-namespace/rule2": prometheusRule2,
			"other-namespace/rule3":  prometheusRule3,
		})

		options := management.Options{
			PrometheusRuleNamespace: "target-namespace",
		}

		rules, err := client.ListRules(ctx, options)

		Expect(err).ToNot(HaveOccurred())
		Expect(rules).To(HaveLen(2))

		alertNames := []string{rules[0].Alert, rules[1].Alert}
		Expect(alertNames).To(ContainElement("NamespaceAlert1"))
		Expect(alertNames).To(ContainElement("NamespaceAlert2"))
		Expect(alertNames).ToNot(ContainElement("OtherNamespaceAlert"))
	})
})
