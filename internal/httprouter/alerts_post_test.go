package httprouter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/machadovilaca/alerts-ui-management/internal/httprouter"
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/testutils"
)

var _ = Describe("CreateUserDefinedAlertRule", func() {
	var (
		router       http.Handler
		mockK8sRules *testutils.MockPrometheusRuleInterface
		mockK8s      *testutils.MockClient
		mockMapper   *testutils.MockMapperClient
	)

	BeforeEach(func() {
		mockK8sRules = &testutils.MockPrometheusRuleInterface{}
		mockK8s = &testutils.MockClient{
			PrometheusRulesFunc: func() k8s.PrometheusRuleInterface {
				return mockK8sRules
			},
		}
		// Default mapper behavior:
		// - IDs computed from alert name
		// - Lookup returns not found (so create proceed)
		mockMapper = &testutils.MockMapperClient{
			GetAlertingRuleIdFunc: func(rule *monitoringv1.Rule) mapper.PrometheusAlertRuleId {
				return mapper.PrometheusAlertRuleId(rule.Alert)
			},
			FindAlertRuleByIdFunc: func(alertRuleId mapper.PrometheusAlertRuleId) (*mapper.PrometheusRuleId, error) {
				return nil, fmt.Errorf("not found")
			},
		}
	})

	Context("create new user defined alert rule", func() {
		It("creates a new rule", func() {
			mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
			router = httprouter.New(mgmt)

			body := map[string]interface{}{
				"alert":                   "cpuHigh",
				"expr":                    "vector(1)",
				"for":                     "5m",
				"labels":                  map[string]string{"severity": "warning"},
				"annotations":             map[string]string{"summary": "cpu high"},
				"prometheusRuleName":      "user-pr",
				"prometheusRuleNamespace": "default",
				// omit groupName to allow management default
			}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/alerting/rules", bytes.NewReader(buf))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusCreated))
			var resp struct {
				Id string `json:"id"`
			}
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp.Id).NotTo(BeEmpty())

			pr, found, err := mockK8sRules.Get(context.Background(), "default", "user-pr")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			allAlerts := []string{}
			for _, g := range pr.Spec.Groups {
				for _, r := range g.Rules {
					allAlerts = append(allAlerts, r.Alert)
				}
			}
			Expect(allAlerts).To(ContainElement("cpuHigh"))
		})
	})

	Context("duplicate existing rule (clone)", func() {
		It("clones a rule", func() {
			// create the rule to copy
			userPR := monitoringv1.PrometheusRule{}
			userPR.Name = "user-pr"
			userPR.Namespace = "default"
			userPR.Spec.Groups = []monitoringv1.RuleGroup{
				{
					Name:  "g1",
					Rules: []monitoringv1.Rule{{Alert: "u1", Expr: intstr.FromString("vector(1)")}},
				},
			}
			mockK8sRules.SetPrometheusRules(map[string]*monitoringv1.PrometheusRule{
				"default/user-pr": &userPR,
			})

			mockMapper = &testutils.MockMapperClient{
				GetAlertingRuleIdFunc: func(rule *monitoringv1.Rule) mapper.PrometheusAlertRuleId {
					return mapper.PrometheusAlertRuleId(rule.Alert)
				},
				FindAlertRuleByIdFunc: func(alertRuleId mapper.PrometheusAlertRuleId) (*mapper.PrometheusRuleId, error) {
					if string(alertRuleId) == "u1" {
						pr := mapper.PrometheusRuleId{
							Namespace: "default",
							Name:      "user-pr",
						}
						return &pr, nil
					}
					return nil, fmt.Errorf("not found")
				},
			}

			mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
			router = httprouter.New(mgmt)

			body := map[string]interface{}{
				"sourceRuleId":            "u1",
				"alert":                   "u1copy",
				"prometheusRuleName":      "user-pr",
				"prometheusRuleNamespace": "default",
			}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/alerting/rules", bytes.NewReader(buf))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusCreated))
			var resp struct {
				Id string `json:"id"`
			}
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp.Id).NotTo(BeEmpty())

			pr, found, err := mockK8sRules.Get(context.Background(), "default", "user-pr")
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(BeTrue())
			allAlerts := []string{}
			for _, g := range pr.Spec.Groups {
				for _, r := range g.Rules {
					allAlerts = append(allAlerts, r.Alert)
				}
			}
			Expect(allAlerts).To(ContainElements("u1", "u1copy"))
		})
	})

	Context("invalid JSON body", func() {
		It("fails for invalid JSON", func() {
			mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
			router = httprouter.New(mgmt)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/alerting/rules", bytes.NewBufferString("{"))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("invalid request body"))
		})
	})

	Context("missing target PrometheusRule (name/namespace)", func() {
		It("fails for missing target PR", func() {
			mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
			router = httprouter.New(mgmt)

			body := map[string]interface{}{
				"alert": "x",
				"expr":  "vector(1)",
				// missing PR name/namespace
			}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/alerting/rules", bytes.NewReader(buf))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("PrometheusRule Name and Namespace must be specified"))
		})
	})

	Context("target is platform-managed PR", func() {
		It("fails for platform PR", func() {
			mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
			router = httprouter.New(mgmt)

			body := map[string]interface{}{
				"alert":                   "x",
				"expr":                    "vector(1)",
				"prometheusRuleName":      "platform-pr",
				"prometheusRuleNamespace": "openshift-monitoring",
			}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/alerting/rules", bytes.NewReader(buf))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusMethodNotAllowed))
			Expect(w.Body.String()).To(ContainSubstring("cannot add user-defined alert rule to a platform-managed PrometheusRule"))
		})
	})
})
