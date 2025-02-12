package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestService_BeforeCreate(t *testing.T) {
	service := &Service{
		Name:          "test-service",
		Attributes:    Attributes{"key": []string{"value"}},
		Resources:     JSON{"cpu": float64(2), "memory": "4GB"},
		AgentID:       uuid.New(),
		ServiceTypeID: uuid.New(),
	}

	err := service.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.Equal(t, ServiceNew, service.State)
	assert.NotNil(t, service.GormAttributes)
	assert.NotNil(t, service.GormResources)
}

func TestService_AfterFind(t *testing.T) {
	attrs, _ := Attributes{"key": []string{"value"}}.ToGormAttributes()
	resources, _ := JSON{"cpu": float64(2), "memory": "4GB"}.ToGormJSON()

	service := &Service{
		Name:           "test-service",
		State:          ServiceCreated,
		GormAttributes: attrs,
		GormResources:  resources,
	}

	err := service.AfterFind(nil)
	assert.NoError(t, err)
	assert.Equal(t, Attributes{"key": []string{"value"}}, service.Attributes)
	assert.Equal(t, JSON{"cpu": float64(2), "memory": "4GB"}, service.Resources)
}

func TestService_BeforeSave(t *testing.T) {
	service := &Service{
		Name:       "test-service",
		State:      ServiceCreated,
		Attributes: Attributes{"key": []string{"value"}},
		Resources:  JSON{"cpu": float64(2), "memory": "4GB"},
	}

	err := service.BeforeSave(nil)
	assert.NoError(t, err)
	assert.NotNil(t, service.GormAttributes)
	assert.NotNil(t, service.GormResources)
}
