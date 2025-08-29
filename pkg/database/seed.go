package database

import (
	"context"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

func Seed(db *gorm.DB) error {
	tokenRepo := NewTokenRepository(db)

	ctx := context.Background()

	// Create a default admin token
	// The token expires in 1 day and must be changed and extended
	adminTokenID := uuid.New()
	exists, err := tokenRepo.Exists(ctx, adminTokenID)
	if err != nil {
		return err
	}
	if !exists {
		// Use a fixed token value for tests
		const adminTokenValue = "change-me"
		adminToken := &domain.Token{
			BaseEntity: domain.BaseEntity{
				ID: adminTokenID,
			},
			Name:        "Admin Token",
			PlainValue:  adminTokenValue,
			HashedValue: domain.HashTokenValue(adminTokenValue),
			Role:        auth.RoleAdmin,
			ExpireAt:    time.Now().AddDate(0, 0, 1),
		}
		if err := tokenRepo.Create(ctx, adminToken); err != nil {
			return err
		}
	}

	return nil
}
