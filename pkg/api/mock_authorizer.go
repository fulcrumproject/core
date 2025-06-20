package api

import (
	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/domain"
)

type MockAuthorizer struct {
	ShouldSucceed bool
}

var _ auth.Authorizer = (*MockAuthorizer)(nil)

func (m *MockAuthorizer) Authorize(identity *auth.Identity, action auth.Action, object auth.ObjectType, scope auth.ObjectScope) error {
	if !m.ShouldSucceed {
		return domain.NewUnauthorizedErrorf("mock authorization failed")
	}
	return nil
}
