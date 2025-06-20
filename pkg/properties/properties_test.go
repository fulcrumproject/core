package properties

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUUID(t *testing.T) {
	t.Run("NewUUID", func(t *testing.T) {
		// Test that NewUUID generates a non-nil UUID
		id := NewUUID()
		assert.NotEqual(t, uuid.Nil, id)
	})

	t.Run("ParseUUID", func(t *testing.T) {
		tests := []struct {
			name    string
			input   string
			wantErr bool
		}{
			{
				name:    "Valid UUID",
				input:   "f47ac10b-58cc-0372-8567-0e02b2c3d479",
				wantErr: false,
			},
			{
				name:    "Invalid UUID format",
				input:   "not-a-uuid",
				wantErr: true,
			},
			{
				name:    "Empty string",
				input:   "",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				id, err := ParseUUID(tt.input)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.input, id.String())
				}
			})
		}
	})
}
