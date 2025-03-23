package domain

import (
	"database/sql/driver"
	"encoding/json"
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

func TestCountryCode(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name       string
			code       CountryCode
			wantErr    bool
			errMessage string
		}{
			{
				name:    "Valid code",
				code:    "US",
				wantErr: false,
			},
			{
				name:       "Too short",
				code:       "A",
				wantErr:    true,
				errMessage: "invalid lentgh",
			},
			{
				name:       "Too long",
				code:       "USA",
				wantErr:    true,
				errMessage: "invalid lentgh",
			},
			{
				name:       "Lowercase letters",
				code:       "us",
				wantErr:    true,
				errMessage: "invalid chars",
			},
			{
				name:       "Contains numbers",
				code:       "U1",
				wantErr:    true,
				errMessage: "invalid chars",
			},
			{
				name:       "Empty string",
				code:       "",
				wantErr:    true,
				errMessage: "invalid lentgh",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.code.Validate()
				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})

	t.Run("ParseCountryCode", func(t *testing.T) {
		tests := []struct {
			name       string
			input      string
			wantErr    bool
			errMessage string
		}{
			{
				name:    "Valid code",
				input:   "US",
				wantErr: false,
			},
			{
				name:       "Invalid code",
				input:      "USA",
				wantErr:    true,
				errMessage: "invalid lentgh",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				code, err := ParseCountryCode(tt.input)
				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
				} else {
					assert.NoError(t, err)
					assert.Equal(t, CountryCode(tt.input), code)
				}
			})
		}
	})
}

func TestAttributes(t *testing.T) {
	t.Run("Scan", func(t *testing.T) {
		tests := []struct {
			name    string
			input   interface{}
			want    Attributes
			wantErr bool
		}{
			{
				name:  "Valid JSON bytes",
				input: []byte(`{"key": ["value1", "value2"]}`),
				want:  Attributes{"key": {"value1", "value2"}},
			},
			{
				name:    "Invalid JSON bytes",
				input:   []byte(`{"key": not-valid-json}`),
				wantErr: true,
			},
			{
				name:  "Nil input",
				input: nil,
				want:  Attributes{},
			},
			{
				name:    "Non-bytes input",
				input:   123,
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var a Attributes
				err := a.Scan(tt.input)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.want, a)
				}
			})
		}
	})

	t.Run("Value", func(t *testing.T) {
		tests := []struct {
			name      string
			attrs     Attributes
			wantValue driver.Value
			wantErr   bool
		}{
			{
				name:      "Valid attributes",
				attrs:     Attributes{"key": {"value1", "value2"}},
				wantValue: []byte(`{"key":["value1","value2"]}`),
			},
			{
				name:      "Empty attributes",
				attrs:     Attributes{},
				wantValue: []byte(`{}`),
			},
			{
				name:      "Nil attributes",
				attrs:     nil,
				wantValue: nil,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				val, err := tt.attrs.Value()
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantValue, val)
				}
			})
		}
	})

	t.Run("GormDataType", func(t *testing.T) {
		attrs := Attributes{}
		assert.Equal(t, "jsonb", attrs.GormDataType())
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		tests := []struct {
			name      string
			attrs     Attributes
			wantValue string
			wantErr   bool
		}{
			{
				name:      "Valid attributes",
				attrs:     Attributes{"key": {"value1", "value2"}},
				wantValue: `{"key":["value1","value2"]}`,
			},
			{
				name:      "Empty attributes",
				attrs:     Attributes{},
				wantValue: `{}`,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				bytes, err := json.Marshal(tt.attrs)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.wantValue, string(bytes))
				}
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			name      string
			jsonValue string
			want      Attributes
			wantErr   bool
		}{
			{
				name:      "Valid JSON",
				jsonValue: `{"key":["value1","value2"]}`,
				want:      Attributes{"key": {"value1", "value2"}},
			},
			{
				name:      "Empty JSON object",
				jsonValue: `{}`,
				want:      Attributes{},
			},
			{
				name:      "Invalid JSON",
				jsonValue: `{"key":not-valid-json}`,
				wantErr:   true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var a Attributes
				err := json.Unmarshal([]byte(tt.jsonValue), &a)
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Equal(t, tt.want, a)
				}
			})
		}
	})

	t.Run("Validate", func(t *testing.T) {
		tests := []struct {
			name       string
			attrs      Attributes
			wantErr    bool
			errMessage string
		}{
			{
				name:    "Valid attributes",
				attrs:   Attributes{"key": {"value1", "value2"}},
				wantErr: false,
			},
			{
				name:       "Empty key",
				attrs:      Attributes{"": {"value"}},
				wantErr:    true,
				errMessage: "keys cannot be empty",
			},
			{
				name:       "Empty values array",
				attrs:      Attributes{"key": {}},
				wantErr:    true,
				errMessage: "has empty values array",
			},
			{
				name:       "Empty value in array",
				attrs:      Attributes{"key": {"value1", ""}},
				wantErr:    true,
				errMessage: "has an empty value",
			},
			{
				name:    "Empty attributes",
				attrs:   Attributes{},
				wantErr: false,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := tt.attrs.Validate()
				if tt.wantErr {
					assert.Error(t, err)
					if tt.errMessage != "" {
						assert.Contains(t, err.Error(), tt.errMessage)
					}
				} else {
					assert.NoError(t, err)
				}
			})
		}
	})
}
