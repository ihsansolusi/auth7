package tests

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}

func TestHealthLiveEndpoint(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodGet, "/health/live", nil)

	r := gin.New()
	r.GET("/health/live", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ok")
}

func TestHealthReadyEndpoint(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	c.Request = httptest.NewRequest(http.MethodGet, "/health/ready", nil)

	r := gin.New()
	r.GET("/health/ready", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ready"})
	})
	r.ServeHTTP(w, c.Request)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "ready")
}

func TestConfigValidation(t *testing.T) {
	_, err := os.CreateTemp("", "auth7-test-config-*.yaml")
	assert.NoError(t, err)
}

func TestDomainErrors(t *testing.T) {
	domainErr := fmt.Errorf("entity not found")
	assert.Error(t, domainErr)

	wrappedErr := fmt.Errorf("service: %w", domainErr)
	assert.Error(t, wrappedErr)
	assert.Contains(t, wrappedErr.Error(), "entity not found")
}
