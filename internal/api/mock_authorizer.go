package api

import (
	"context"

	"fulcrumproject.org/core/internal/domain"
)

type MockAuthorizer struct {
	ShouldSucceed bool
}

var _ domain.Authorizer = (*MockAuthorizer)(nil)

func (m *MockAuthorizer) Authorize(identity domain.AuthIdentity, subject domain.AuthSubject, action domain.AuthAction, scope *domain.AuthTargetScope) error {
	if !m.ShouldSucceed {
		return domain.NewUnauthorizedErrorf("mock authorization failed")
	}
	return nil
}

func (m *MockAuthorizer) AuthorizeCtx(ctx context.Context, subject domain.AuthSubject, action domain.AuthAction, scope *domain.AuthTargetScope) error {
	if !m.ShouldSucceed {
		return domain.NewUnauthorizedErrorf("mock authorization failed")
	}
	return nil
}
