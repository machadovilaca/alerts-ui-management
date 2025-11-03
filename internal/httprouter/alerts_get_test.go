package httprouter_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/machadovilaca/alerts-ui-management/internal/httprouter"
	"github.com/machadovilaca/alerts-ui-management/pkg/k8s"
	"github.com/machadovilaca/alerts-ui-management/pkg/management"
	"github.com/machadovilaca/alerts-ui-management/pkg/management/testutils"
)

var _ = Describe("GetAlerts", func() {
	var (
		mockK8s              *testutils.MockClient
		mockPrometheusAlerts *testutils.MockPrometheusAlertsInterface
		mockManagement       management.Client
		router               http.Handler
	)

	BeforeEach(func() {
		By("setting up mock clients")
		mockPrometheusAlerts = &testutils.MockPrometheusAlertsInterface{}
		mockK8s = &testutils.MockClient{
			PrometheusAlertsFunc: func() k8s.PrometheusAlertsInterface {
				return mockPrometheusAlerts
			},
		}

		mockManagement = management.NewWithCustomMapper(context.Background(), mockK8s, &testutils.MockMapperClient{})
		router = httprouter.New(mockK8s, mockManagement)
	})

	Context("when getting all alerts without filters", func() {
		It("should return all active alerts", func() {
			By("setting up test alerts")
			testAlerts := []k8s.ActiveAlert{
				{
					Name:     "HighCPUUsage",
					Severity: "warning",
					Labels: map[string]string{
						"alertname": "HighCPUUsage",
						"severity":  "warning",
						"namespace": "default",
					},
					Annotations: map[string]string{
						"description": "CPU usage is high",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
				{
					Name:     "LowMemory",
					Severity: "critical",
					Labels: map[string]string{
						"alertname": "LowMemory",
						"severity":  "critical",
						"namespace": "monitoring",
					},
					Annotations: map[string]string{
						"description": "Memory is running low",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
			}
			mockPrometheusAlerts.SetActiveAlerts(testAlerts)

			By("making the request")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying the response")
			Expect(w.Code).To(Equal(http.StatusOK))
			Expect(w.Header().Get("Content-Type")).To(Equal("application/json"))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(2))
			Expect(response.Alerts[0].Name).To(Equal("HighCPUUsage"))
			Expect(response.Alerts[1].Name).To(Equal("LowMemory"))
		})

		It("should return empty array when no alerts exist", func() {
			By("setting up empty alerts")
			mockPrometheusAlerts.SetActiveAlerts([]k8s.ActiveAlert{})

			By("making the request")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying the response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(BeEmpty())
		})
	})

	Context("when filtering alerts by labels", func() {
		BeforeEach(func() {
			By("setting up test alerts with various labels")
			testAlerts := []k8s.ActiveAlert{
				{
					Name:     "HighCPUUsage",
					Severity: "warning",
					Labels: map[string]string{
						"alertname": "HighCPUUsage",
						"severity":  "warning",
						"namespace": "default",
						"service":   "api",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
				{
					Name:     "LowMemory",
					Severity: "critical",
					Labels: map[string]string{
						"alertname": "LowMemory",
						"severity":  "critical",
						"namespace": "default",
						"service":   "database",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
				{
					Name:     "DiskSpaceLow",
					Severity: "warning",
					Labels: map[string]string{
						"alertname": "DiskSpaceLow",
						"severity":  "warning",
						"namespace": "monitoring",
						"service":   "storage",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
			}
			mockPrometheusAlerts.SetActiveAlerts(testAlerts)
		})

		It("should filter alerts by single label", func() {
			By("making request with severity filter")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[severity]=warning", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying filtered response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(2))
			Expect(response.Alerts[0].Severity).To(Equal("warning"))
			Expect(response.Alerts[1].Severity).To(Equal("warning"))
		})

		It("should filter alerts by multiple labels", func() {
			By("making request with multiple label filters")
			params := url.Values{}
			params.Add("labels[severity]", "warning")
			params.Add("labels[namespace]", "default")

			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?"+params.Encode(), nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying filtered response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(1))
			Expect(response.Alerts[0].Name).To(Equal("HighCPUUsage"))
			Expect(response.Alerts[0].Labels["severity"]).To(Equal("warning"))
			Expect(response.Alerts[0].Labels["namespace"]).To(Equal("default"))
		})

		It("should return empty array when no alerts match the filter", func() {
			By("making request with non-matching filter")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[severity]=info", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying empty response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(BeEmpty())
		})

		It("should filter alerts when label value doesn't match", func() {
			By("making request with specific namespace filter")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[namespace]=monitoring", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying filtered response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(1))
			Expect(response.Alerts[0].Name).To(Equal("DiskSpaceLow"))
		})

		It("should filter alerts when label key doesn't exist", func() {
			By("making request with non-existent label key")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[nonexistent]=value", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying empty response")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(BeEmpty())
		})
	})

	Context("when handling errors", func() {
		It("should return 500 when GetActiveAlerts fails", func() {
			By("configuring mock to return error")
			mockPrometheusAlerts.GetActiveAlertsFunc = func(ctx context.Context) ([]k8s.ActiveAlert, error) {
				return nil, fmt.Errorf("connection error")
			}

			By("making the request")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying error response")
			Expect(w.Code).To(Equal(http.StatusInternalServerError))
			Expect(w.Body.String()).To(ContainSubstring("Failed to get alerts"))
			Expect(w.Body.String()).To(ContainSubstring("connection error"))
		})
	})

	Context("when dealing with edge cases", func() {
		It("should handle alerts with nil labels map", func() {
			By("setting up alert with nil labels")
			testAlerts := []k8s.ActiveAlert{
				{
					Name:     "NoLabelsAlert",
					Severity: "warning",
					Labels:   nil,
					State:    "firing",
					ActiveAt: time.Now(),
				},
			}
			mockPrometheusAlerts.SetActiveAlerts(testAlerts)

			By("making request with label filter")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[severity]=warning", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying it's filtered out correctly")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(BeEmpty())
		})

		It("should handle alerts with empty labels map", func() {
			By("setting up alert with empty labels")
			testAlerts := []k8s.ActiveAlert{
				{
					Name:     "EmptyLabelsAlert",
					Severity: "warning",
					Labels:   map[string]string{},
					State:    "firing",
					ActiveAt: time.Now(),
				},
			}
			mockPrometheusAlerts.SetActiveAlerts(testAlerts)

			By("making request without filters")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying alert is returned")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(1))
		})

		It("should handle empty label filter values", func() {
			By("setting up test alerts")
			testAlerts := []k8s.ActiveAlert{
				{
					Name:     "TestAlert",
					Severity: "warning",
					Labels: map[string]string{
						"severity":  "",
						"namespace": "default",
					},
					State:    "firing",
					ActiveAt: time.Now(),
				},
			}
			mockPrometheusAlerts.SetActiveAlerts(testAlerts)

			By("making request with empty label value filter")
			req := httptest.NewRequest(http.MethodGet, "/api/v1/alerting/alerts?labels[severity]=", nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			By("verifying exact match with empty value")
			Expect(w.Code).To(Equal(http.StatusOK))

			var response httprouter.GetAlertsResponse
			err := json.NewDecoder(w.Body).Decode(&response)
			Expect(err).NotTo(HaveOccurred())
			Expect(response.Alerts).To(HaveLen(1))
		})
	})
})
