package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestServiceGroup_Creation(t *testing.T) {
	group := &ServiceGroup{
		Name: "test-group",
	}

	// Test that BaseEntity fields are properly initialized
	err := group.BeforeCreate(nil)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, group.ID)

	// Test relationships
	service := &Service{
		Name:    "test-service",
		State:   ServiceNew,
		GroupID: group.ID,
	}
	group.Services = append(group.Services, *service)
	assert.Equal(t, 1, len(group.Services))
	assert.Equal(t, group.ID, group.Services[0].GroupID)
}
