package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fulcrumproject/core/pkg/auth"
	"github.com/fulcrumproject/core/pkg/authz"
	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/properties"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewInfrastructureHandler(t *testing.T) {
	querier := domain.NewMockInfrastructureQuerier(t)
	commander := domain.NewMockInfrastructureCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewInfrastructureHandler(querier, commander, authz)
	assert.NotNil(t, handler)
	assert.Equal(t, querier, handler.querier)
	assert.Equal(t, commander, handler.commander)
	assert.Equal(t, authz, handler.authz)
}

func TestInfrastructureHandlerRoutes(t *testing.T) {
	querier := domain.NewMockInfrastructureQuerier(t)
	commander := domain.NewMockInfrastructureCommander(t)
	authz := authz.NewMockAuthorizer(t)

	handler := NewInfrastructureHandler(querier, commander, authz)
	routeFunc := handler.Routes()
	assert.NotNil(t, routeFunc)

	r := chi.NewRouter()
	routeFunc(r)

	walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		switch {
		case method == "GET" && route == "/":
		case method == "POST" && route == "/":
		case method == "GET" && route == "/{id}":
		case method == "PATCH" && route == "/{id}":
		case method == "DELETE" && route == "/{id}":
		case method == "GET" && route == "/me":
		default:
			return fmt.Errorf("unexpected route: %s %s", method, route)
		}
		return nil
	}
	assert.NoError(t, chi.Walk(r, walkFunc))
}

func TestInfrastructureHandlerCreate(t *testing.T) {
	commander := domain.NewMockInfrastructureCommander(t)
	handler := &InfrastructureHandler{commander: commander}

	providerID := properties.UUID(uuid.New())
	infraTypeID := properties.UUID(uuid.New())
	req := &CreateInfrastructureReq{
		Name:                 "csi-node-01",
		ProviderID:           providerID,
		InfrastructureTypeID: infraTypeID,
		Tags:                 []string{"region:eu"},
	}

	commander.EXPECT().
		Create(mock.Anything, mock.MatchedBy(func(params domain.CreateInfrastructureParams) bool {
			return params.Name == "csi-node-01" &&
				params.ProviderID == providerID &&
				params.InfrastructureTypeID == infraTypeID &&
				len(params.Tags) == 1 && params.Tags[0] == "region:eu"
		})).
		Return(&domain.Infrastructure{
			BaseEntity:           domain.BaseEntity{ID: properties.UUID(uuid.New())},
			Name:                 "csi-node-01",
			ProviderID:           providerID,
			InfrastructureTypeID: infraTypeID,
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   properties.UUID(uuid.New()),
		Name: "admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Create(ctx, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestInfrastructureHandlerUpdate(t *testing.T) {
	commander := domain.NewMockInfrastructureCommander(t)
	handler := &InfrastructureHandler{commander: commander}

	id := properties.UUID(uuid.New())
	name := "Renamed"
	req := &UpdateInfrastructureReq{Name: &name}

	commander.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(params domain.UpdateInfrastructureParams) bool {
			return params.ID == id && params.Name != nil && *params.Name == "Renamed"
		})).
		Return(&domain.Infrastructure{
			BaseEntity: domain.BaseEntity{ID: id},
			Name:       "Renamed",
		}, nil)

	ctx := auth.WithIdentity(context.Background(), &auth.Identity{
		ID:   properties.UUID(uuid.New()),
		Name: "admin",
		Role: auth.RoleAdmin,
	})
	result, err := handler.Update(ctx, id, req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

// TestHandleGetMe_Infrastructure synthesizes a RoleAgent identity whose
// Scope.AgentID carries the infrastructure id (per the Phase 2 plan: the
// AgentID coordinate doubles as the self-reference for Infrastructure).
func TestHandleGetMe_Infrastructure(t *testing.T) {
	testCases := []struct {
		name           string
		setup          func(*domain.MockInfrastructureQuerier, properties.UUID)
		expectedStatus int
	}{
		{
			name: "Success",
			setup: func(q *domain.MockInfrastructureQuerier, infraID properties.UUID) {
				q.EXPECT().
					Get(mock.Anything, infraID).
					Return(&domain.Infrastructure{
						BaseEntity:           domain.BaseEntity{ID: infraID},
						Name:                 "csi-node-01",
						ProviderID:           properties.UUID(uuid.New()),
						InfrastructureTypeID: properties.UUID(uuid.New()),
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "NotFound",
			setup: func(q *domain.MockInfrastructureQuerier, infraID properties.UUID) {
				q.EXPECT().
					Get(mock.Anything, infraID).
					Return(nil, domain.NewNotFoundErrorf("infrastructure not found"))
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			infraID := properties.UUID(uuid.New())
			querier := domain.NewMockInfrastructureQuerier(t)
			commander := domain.NewMockInfrastructureCommander(t)
			mockAuthz := authz.NewMockAuthorizer(t)
			tc.setup(querier, infraID)

			handler := NewInfrastructureHandler(querier, commander, mockAuthz)

			req := httptest.NewRequest("GET", "/infrastructures/me", nil)
			req = req.WithContext(auth.WithIdentity(req.Context(), newMockAuthAgentWithID(infraID)))

			w := httptest.NewRecorder()
			handler.GetMe(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
		})
	}
}

func TestInfrastructureToRes(t *testing.T) {
	createdAt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	id := properties.UUID(uuid.New())
	providerID := properties.UUID(uuid.New())
	infraTypeID := properties.UUID(uuid.New())

	infra := &domain.Infrastructure{
		BaseEntity: domain.BaseEntity{
			ID:        id,
			CreatedAt: createdAt,
			UpdatedAt: updatedAt,
		},
		Name:                 "csi-node-01",
		ProviderID:           providerID,
		InfrastructureTypeID: infraTypeID,
		Tags:                 []string{"region:eu"},
		Configuration:        &properties.JSON{"endpoint": "https://x"},
	}

	res := InfrastructureToRes(infra)
	assert.Equal(t, id, res.ID)
	assert.Equal(t, "csi-node-01", res.Name)
	assert.Equal(t, providerID, res.ProviderID)
	assert.Equal(t, infraTypeID, res.InfrastructureTypeID)
	assert.Equal(t, []string{"region:eu"}, res.Tags)
	assert.Equal(t, infra.Configuration, res.Configuration)
	assert.Equal(t, JSONUTCTime(createdAt), res.CreatedAt)
	assert.Equal(t, JSONUTCTime(updatedAt), res.UpdatedAt)
}

func TestInfrastructureToRes_NilConfiguration(t *testing.T) {
	infra := &domain.Infrastructure{
		BaseEntity:           domain.BaseEntity{ID: properties.UUID(uuid.New())},
		Name:                 "csi-node-02",
		ProviderID:           properties.UUID(uuid.New()),
		InfrastructureTypeID: properties.UUID(uuid.New()),
	}
	res := InfrastructureToRes(infra)
	assert.Nil(t, res.Configuration)
}
