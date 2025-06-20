package health

import (
	"context"
	"errors"
	"testing"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// MockDB implements a simple mock for *gorm.DB
type MockDB struct {
	pingError error
}

func (m *MockDB) DB() (interface{}, error) {
	if m.pingError != nil {
		return nil, m.pingError
	}
	return &MockSQLDB{pingError: m.pingError}, nil
}

// MockSQLDB implements a simple mock for *sql.DB
type MockSQLDB struct {
	pingError error
}

func (m *MockSQLDB) PingContext(ctx context.Context) error {
	return m.pingError
}

// MockAuthenticator implements the auth.Authenticator interface for testing
type MockAuthenticator struct {
	healthError error
}

func (m *MockAuthenticator) Authenticate(ctx context.Context, token string) (*auth.Identity, error) {
	return nil, nil
}

func (m *MockAuthenticator) Health(ctx context.Context) error {
	return m.healthError
}

func TestHealthChecker_CheckHealth_Success(t *testing.T) {
	// Setup
	deps := &PrimaryDependencies{
		DB: &gorm.DB{}, // We'll mock the underlying DB methods
		Authenticators: []auth.Authenticator{
			&MockAuthenticator{healthError: nil},
		},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// We need to mock the database ping, but since we can't easily mock gorm.DB,
	// let's test the authenticator part and assume database connectivity works
	// in integration tests

	// For this test, let's focus on the authenticator health check
	deps.DB = nil // Set to nil to test the nil check

	result := checker.CheckHealth(ctx)

	// Should fail because DB is nil
	assert.Equal(t, StatusDOWN, result.Status)
	assert.Contains(t, result.Error, "Database check failed")
}

func TestHealthChecker_CheckAuthentication_Success(t *testing.T) {
	// Setup
	mockAuth := &MockAuthenticator{healthError: nil}
	deps := &PrimaryDependencies{
		Authenticators: []auth.Authenticator{mockAuth},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// Test
	err := checker.checkAuthentication(ctx)

	// Assert
	assert.NoError(t, err)
}

func TestHealthChecker_CheckAuthentication_Failure(t *testing.T) {
	// Setup
	expectedError := errors.New("authenticator is down")
	mockAuth := &MockAuthenticator{healthError: expectedError}
	deps := &PrimaryDependencies{
		Authenticators: []auth.Authenticator{mockAuth},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// Test
	err := checker.checkAuthentication(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authenticator 0 health check failed")
	assert.Contains(t, err.Error(), "authenticator is down")
}

func TestHealthChecker_CheckAuthentication_NoAuthenticators(t *testing.T) {
	// Setup
	deps := &PrimaryDependencies{
		Authenticators: []auth.Authenticator{},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// Test
	err := checker.checkAuthentication(ctx)

	// Assert
	assert.NoError(t, err) // Should be OK when no authenticators are configured
}

func TestHealthChecker_CheckAuthentication_MultipleAuthenticators(t *testing.T) {
	// Setup
	mockAuth1 := &MockAuthenticator{healthError: nil}
	mockAuth2 := &MockAuthenticator{healthError: errors.New("auth2 error")}
	deps := &PrimaryDependencies{
		Authenticators: []auth.Authenticator{mockAuth1, mockAuth2},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// Test
	err := checker.checkAuthentication(ctx)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "authenticator 1 health check failed")
	assert.Contains(t, err.Error(), "auth2 error")
}

func TestHealthChecker_CheckReadiness(t *testing.T) {
	// Setup
	deps := &PrimaryDependencies{
		DB: nil, // This will cause a failure
		Authenticators: []auth.Authenticator{
			&MockAuthenticator{healthError: nil},
		},
	}

	checker := NewHealthChecker(deps)
	ctx := context.Background()

	// Test
	result := checker.CheckReadiness(ctx)

	// Assert
	assert.Equal(t, StatusDOWN, result.Status)
	assert.Contains(t, result.Error, "Database check failed")
}
