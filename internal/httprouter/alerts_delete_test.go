package httprouter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	"github.com/machadovilaca/alerts-ui-management/internal/httprouter"
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/mapper"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/testutils"
)

var _ = Describe("BulkDeleteUserDefinedAlertRules", func() {
	var (
		router       http.Handler
		mockK8sRules *testutils.MockPrometheusRuleInterface
		mockK8s      *testutils.MockClient
		mockMapper   *testutils.MockMapperClient
	)

	BeforeEach(func() {
		mockK8sRules = &testutils.MockPrometheusRuleInterface{}

		userPR := monitoringv1.PrometheusRule{}
		userPR.Name = "user-pr"
		userPR.Namespace = "default"
		userPR.Spec.Groups = []monitoringv1.RuleGroup{
			{
				Name:  "g1",
				Rules: []monitoringv1.Rule{{Alert: "u1"}, {Alert: "u2"}},
			},
		}

		platformPR := monitoringv1.PrometheusRule{}
		platformPR.Name = "openshift-platform-pr"
		platformPR.Namespace = "default"
		platformPR.Spec.Groups = []monitoringv1.RuleGroup{
			{
				Name:  "pg1",
				Rules: []monitoringv1.Rule{{Alert: "platform1"}},
			},
		}

		mockK8sRules.SetPrometheusRules(map[string]*monitoringv1.PrometheusRule{
			"default/user-pr":               &userPR,
			"default/openshift-platform-pr": &platformPR,
		})

		mockK8s = &testutils.MockClient{
			PrometheusRulesFunc: func() k8s.PrometheusRuleInterface {
				return mockK8sRules
			},
		}

		mockMapper = &testutils.MockMapperClient{
			GetAlertingRuleIdFunc: func(rule *monitoringv1.Rule) mapper.PrometheusAlertRuleId {
				return mapper.PrometheusAlertRuleId(rule.Alert)
			},
			FindAlertRuleByIdFunc: func(alertRuleId mapper.PrometheusAlertRuleId) (*mapper.PrometheusRuleId, *mapper.AlertRelabelConfigId, error) {
				id := string(alertRuleId)
				pr := mapper.PrometheusRuleId{
					Namespace: "default",
					Name:      "user-pr",
				}
				if id == "platform1" {
					pr.Name = "openshift-platform-pr"
				}
				return &pr, nil, nil
			},
		}

		mgmt := management.NewWithCustomMapper(context.Background(), mockK8s, mockMapper)
		router = httprouter.New(mockK8s, mgmt)
	})

	Context("when deleting multiple rules", func() {
		It("returns per-id results and applies side-effects", func() {
			body := map[string]interface{}{"ruleIds": []string{"u1", "platform1", ""}}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerting/rules", bytes.NewReader(buf))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusOK))
			var resp struct {
				DeletedIds []string          `json:"deletedIds"`
				Failed     map[string]string `json:"failed"`
			}
			Expect(json.NewDecoder(w.Body).Decode(&resp)).To(Succeed())
			Expect(resp.DeletedIds).To(ContainElements("u1"))
			Expect(resp.Failed).To(HaveKey("platform1"))
			Expect(resp.Failed["platform1"]).To(ContainSubstring("can't delete platform alert"))
			Expect(resp.Failed).To(HaveKey(""))
			Expect(resp.Failed[""]).To(ContainSubstring("failed to delete user alert"))

			prUser, err := mockK8sRules.Get(context.Background(), "default", "user-pr")
			Expect(err).NotTo(HaveOccurred())
			userRuleNames := []string{}
			for _, g := range prUser.Spec.Groups {
				for _, r := range g.Rules {
					userRuleNames = append(userRuleNames, r.Alert)
				}
			}
			Expect(userRuleNames).NotTo(ContainElement("u1"))
			Expect(userRuleNames).To(ContainElement("u2"))

			prPlatform, err := mockK8sRules.Get(context.Background(), "default", "openshift-platform-pr")
			Expect(err).NotTo(HaveOccurred())
			foundPlatform := false
			for _, g := range prPlatform.Spec.Groups {
				for _, r := range g.Rules {
					if r.Alert == "platform1" {
						foundPlatform = true
					}
				}
			}
			Expect(foundPlatform).To(BeTrue())
		})
	})

	Context("when request body is invalid", func() {
		It("returns 400", func() {
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerting/rules", bytes.NewBufferString("{"))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("invalid request body"))
		})
	})

	Context("when ruleIds is empty", func() {
		It("returns 400", func() {
			body := map[string]interface{}{"ruleIds": []string{}}
			buf, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodDelete, "/api/v1/alerting/rules", bytes.NewReader(buf))
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			Expect(w.Code).To(Equal(http.StatusBadRequest))
			Expect(w.Body.String()).To(ContainSubstring("ruleIds is required"))
		})
	})
})
