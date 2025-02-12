package domain

import (
	"testing"

	"github.com/google/uuid"
)

func TestValidateUUID(t *testing.T) {
	tests := []struct {
		name    string
		id      UUID
		wantErr bool
	}{
		{
			name:    "Valid UUID",
			id:      uuid.New(),
			wantErr: false,
		},
		{
			name:    "Nil UUID",
			id:      UUID{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateUUID(tt.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUUID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{
			name:    "Valid name",
			input:   "Test Name",
			wantErr: false,
		},
		{
			name:    "Empty name",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateCountryCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		wantErr bool
	}{
		{
			name:    "Valid country code",
			code:    "US",
			wantErr: false,
		},
		{
			name:    "Invalid country code - lowercase",
			code:    "us",
			wantErr: true,
		},
		{
			name:    "Invalid country code - too long",
			code:    "USA",
			wantErr: true,
		},
		{
			name:    "Invalid country code - too short",
			code:    "U",
			wantErr: true,
		},
		{
			name:    "Invalid country code - empty",
			code:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCountryCode(tt.code)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCountryCode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateAttributes(t *testing.T) {
	tests := []struct {
		name    string
		attrs   Attributes
		wantErr bool
	}{
		{
			name: "Valid attributes",
			attrs: Attributes{
				"key1": {"value1", "value2"},
				"key2": {"value3"},
			},
			wantErr: false,
		},
		{
			name:    "Nil attributes",
			attrs:   nil,
			wantErr: false,
		},
		{
			name: "Empty key",
			attrs: Attributes{
				"": {"value1"},
			},
			wantErr: true,
		},
		{
			name: "Nil values",
			attrs: Attributes{
				"key1": nil,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAttributes(tt.attrs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAttributes() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON
		wantErr bool
	}{
		{
			name: "Valid JSON",
			json: JSON{
				"key1": "value1",
				"key2": 123,
			},
			wantErr: false,
		},
		{
			name:    "Nil JSON",
			json:    nil,
			wantErr: false,
		},
		{
			name: "Empty key",
			json: JSON{
				"": "value1",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateJSON(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
