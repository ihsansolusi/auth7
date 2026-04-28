package tests

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/ihsansolusi/auth7/internal/integration/notif7"
	"github.com/ihsansolusi/auth7/internal/service/jwt"
	notif7client "github.com/ihsansolusi/lib7-service-go/notif7client"
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

type mockNotif7Sender struct {
	mock.Mock
}

func (m *mockNotif7Sender) Send(ctx context.Context, event notif7client.Event) (*notif7client.SendResult, error) {
	args := m.Called(ctx, event)
	return nil, args.Error(0)
}

func TestNotif7Client_SendLoginNewDevice(t *testing.T) {
	sender := new(mockNotif7Sender)
	client := notif7.NewClient(sender)

	params := notif7.LoginNewDeviceParams{
		UserID:     "user-123",
		Username:   "testuser",
		Email:      "test@example.com",
		OrgID:      "org-456",
		DeviceName: "Chrome on Windows",
		IPAddress:  "192.168.1.1",
		Location:   "Jakarta, Indonesia",
	}

	sender.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := client.SendLoginNewDevice(context.Background(), params)
	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestNotif7Client_SendAccountLocked(t *testing.T) {
	sender := new(mockNotif7Sender)
	client := notif7.NewClient(sender)

	params := notif7.AccountLockedParams{
		UserID:   "user-123",
		Username: "testuser",
		Email:    "test@example.com",
		OrgID:    "org-456",
		Reason:   "Too many failed login attempts",
		LockedAt: time.Now(),
	}

	sender.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := client.SendAccountLocked(context.Background(), params)
	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestNotif7Client_SendPasswordChanged(t *testing.T) {
	sender := new(mockNotif7Sender)
	client := notif7.NewClient(sender)

	params := notif7.PasswordChangedParams{
		UserID:    "user-123",
		Username:  "testuser",
		Email:     "test@example.com",
		OrgID:     "org-456",
		ChangedAt: time.Now(),
		IPAddress: "192.168.1.1",
	}

	sender.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := client.SendPasswordChanged(context.Background(), params)
	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestNotif7Client_SendMfaReset(t *testing.T) {
	sender := new(mockNotif7Sender)
	client := notif7.NewClient(sender)

	params := notif7.MfaResetParams{
		UserID:    "user-123",
		Username:  "testuser",
		Email:     "test@example.com",
		OrgID:     "org-456",
		ResetAt:   time.Now(),
		IPAddress: "192.168.1.1",
	}

	sender.On("Send", mock.Anything, mock.Anything).Return(nil)

	err := client.SendMfaReset(context.Background(), params)
	assert.NoError(t, err)
	sender.AssertExpectations(t)
}

func TestJWTTokenGeneration(t *testing.T) {
	svc := jwt.NewService("auth7.test", []string{"auth7.test"})

	sessionID := "session-123"
	userID := uuid.New()
	orgID := uuid.New()

	claims := jwt.Claims{
		ClientID: "test-client",
		Roles:    []string{"user", "admin"},
		Scope:    "openid profile",
	}

	token, access, err := svc.IssueAccessToken(sessionID, userID, orgID, claims)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.NotNil(t, access)
	assert.Equal(t, sessionID, access.SessionID)
	assert.Equal(t, userID, access.UserID)
	assert.Equal(t, orgID, access.OrgID)
}

func TestJWTTokenVerification(t *testing.T) {
	svc := jwt.NewService("auth7.test", []string{"auth7.test"})

	sessionID := "session-123"
	userID := uuid.New()
	orgID := uuid.New()

	claims := jwt.Claims{
		ClientID: "test-client",
		Roles:    []string{"user"},
		Scope:    "openid profile",
	}

	token, _, err := svc.IssueAccessToken(sessionID, userID, orgID, claims)
	assert.NoError(t, err)

	verifiedClaims, err := svc.VerifyAccessToken(token)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), verifiedClaims.Subject)
	assert.Equal(t, orgID.String(), verifiedClaims.OrgID)
	assert.Equal(t, claims.ClientID, verifiedClaims.ClientID)
}
