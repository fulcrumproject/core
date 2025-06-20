package health

import (
	"context"
	"fmt"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"gorm.io/gorm"
)

// Status represents the health status
type Status string

const (
	StatusUP   Status = "UP"
	StatusDOWN Status = "DOWN"
)

// CheckResult represents the result of a health check
type CheckResult struct {
	Status Status `json:"status"`
	Error  string `json:"error,omitempty"`
}

// Checker defines the interface for health checking
type Checker interface {
	CheckHealth(ctx context.Context) CheckResult
	CheckReadiness(ctx context.Context) CheckResult
}

// PrimaryDependencies holds references to primary dependencies
type PrimaryDependencies struct {
	DB             *gorm.DB
	Authenticators []auth.Authenticator
}

// HealthChecker implements the Checker interface
type HealthChecker struct {
	deps *PrimaryDependencies
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(deps *PrimaryDependencies) *HealthChecker {
	return &HealthChecker{
		deps: deps,
	}
}

// CheckHealth performs health checks on primary dependencies
func (h *HealthChecker) CheckHealth(ctx context.Context) CheckResult {
	return h.checkPrimaryDependencies(ctx)
}

// CheckReadiness performs readiness checks on primary dependencies
func (h *HealthChecker) CheckReadiness(ctx context.Context) CheckResult {
	return h.checkPrimaryDependencies(ctx)
}

// checkPrimaryDependencies checks all primary dependencies
func (h *HealthChecker) checkPrimaryDependencies(ctx context.Context) CheckResult {
	// Check database connectivity
	if err := h.checkDatabase(ctx); err != nil {
		return CheckResult{
			Status: StatusDOWN,
			Error:  fmt.Sprintf("Database check failed: %v", err),
		}
	}

	// Check authentication services
	if err := h.checkAuthentication(ctx); err != nil {
		return CheckResult{
			Status: StatusDOWN,
			Error:  fmt.Sprintf("Authentication check failed: %v", err),
		}
	}

	return CheckResult{
		Status: StatusUP,
	}
}

// checkDatabase verifies database connectivity
func (h *HealthChecker) checkDatabase(ctx context.Context) error {
	if h.deps.DB == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Get the underlying sql.DB to perform a ping
	sqlDB, err := h.deps.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get underlying database: %w", err)
	}

	// Create a context with timeout for the ping
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

// checkAuthentication verifies authentication services are available
func (h *HealthChecker) checkAuthentication(ctx context.Context) error {
	if len(h.deps.Authenticators) == 0 {
		// No authenticators configured - this might be intentional for development
		return nil
	}

	// Check each authenticator's health
	for i, authenticator := range h.deps.Authenticators {
		if err := authenticator.Health(ctx); err != nil {
			return fmt.Errorf("authenticator %d health check failed: %w", i, err)
		}
	}

	return nil
}
