package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMetricsUseRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Middleware())

	router.GET("/track/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(
			http.MethodGet,
			"/track/private-id-123",
			nil,
		),
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status %d", recorder.Code)
	}

	metrics := httptest.NewRecorder()

	Handler().ServeHTTP(
		metrics,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)

	body := metrics.Body.String()

	if metrics.Code != http.StatusOK ||
		!strings.Contains(body, "http_server_requests_total") ||
		!strings.Contains(body, `route="/track/:id"`) ||
		strings.Contains(body, "private-id-123") {
		t.Fatal("metrics did not use bounded route template")
	}
}

func TestMetricsRecordRecoveredPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(Middleware())
	router.Use(gin.Recovery())

	router.GET("/panic-test", func(c *gin.Context) {
		panic("intentional test panic")
	})

	recorder := httptest.NewRecorder()

	router.ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/panic-test", nil),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected HTTP 500, got %d", recorder.Code)
	}

	metrics := httptest.NewRecorder()

	Handler().ServeHTTP(
		metrics,
		httptest.NewRequest(http.MethodGet, "/metrics", nil),
	)

	body := metrics.Body.String()

	if !strings.Contains(body, `route="/panic-test",status_class="5xx"`) {
		t.Fatal("recovered panic was not recorded in metrics")
	}
}
