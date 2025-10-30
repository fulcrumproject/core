package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewVaultEncryption(t *testing.T) {
	tests := []struct {
		name      string
		keySize   int
		expectErr bool
	}{
		{
			name:      "Valid 32-byte key",
			keySize:   32,
			expectErr: false,
		},
		{
			name:      "Invalid 16-byte key",
			keySize:   16,
			expectErr: true,
		},
		{
			name:      "Invalid 0-byte key",
			keySize:   0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := make([]byte, tt.keySize)
			rand.Read(key)

			enc, err := newVaultEncryption(key)

			if tt.expectErr {
				assert.Error(t, err)
				assert.Nil(t, enc)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, enc)
			}
		})
	}
}

func TestVaultEncryptionRoundTrip(t *testing.T) {
	// Create encryption with valid key
	key := make([]byte, 32)
	rand.Read(key)
	enc, err := newVaultEncryption(key)
	require.NoError(t, err)

	tests := []struct {
		name      string
		plaintext []byte
	}{
		{
			name:      "Simple string",
			plaintext: []byte("hello world"),
		},
		{
			name:      "JSON data",
			plaintext: []byte(`{"key":"value","number":123}`),
		},
		{
			name:      "Binary data",
			plaintext: []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := enc.Encrypt(tt.plaintext)
			require.NoError(t, err)
			assert.NotEqual(t, tt.plaintext, ciphertext)

			// Decrypt
			decrypted, err := enc.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestVaultEncryptionDecryptInvalidData(t *testing.T) {
	key := make([]byte, 32)
	rand.Read(key)
	enc, err := newVaultEncryption(key)
	require.NoError(t, err)

	tests := []struct {
		name       string
		ciphertext []byte
	}{
		{
			name:       "Too short",
			ciphertext: []byte{0x01},
		},
		{
			name:       "Empty",
			ciphertext: []byte{},
		},
		{
			name:       "Invalid data",
			ciphertext: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0xFF},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := enc.Decrypt(tt.ciphertext)
			assert.Error(t, err)
		})
	}
}

func TestVaultSaveAndGet(t *testing.T) {
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)

	// Create vault
	key := make([]byte, 32)
	rand.Read(key)
	vault, err := NewVault(tdb.DB, key)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "String value",
			value: "my-secret-password",
		},
		{
			name:  "Number value",
			value: float64(12345),
		},
		{
			name: "Object value",
			value: map[string]any{
				"username": "admin",
				"password": "secret123",
			},
		},
		{
			name:  "Array value",
			value: []any{"secret1", "secret2", "secret3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Generate reference
			refBytes := make([]byte, 16)
			rand.Read(refBytes)
			ref := hex.EncodeToString(refBytes)

			// Save secret
			err := vault.Save(ctx, ref, tt.value, nil)
			require.NoError(t, err)

			// Retrieve secret
			retrieved, err := vault.Get(ctx, ref)
			require.NoError(t, err)
			assert.Equal(t, tt.value, retrieved)
		})
	}
}

func TestVaultGetNotFound(t *testing.T) {
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)

	key := make([]byte, 32)
	rand.Read(key)
	vault, err := NewVault(tdb.DB, key)
	require.NoError(t, err)

	ctx := context.Background()
	_, err = vault.Get(ctx, "nonexistent-reference")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVaultDelete(t *testing.T) {
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)

	key := make([]byte, 32)
	rand.Read(key)
	vault, err := NewVault(tdb.DB, key)
	require.NoError(t, err)

	ctx := context.Background()

	// Generate reference
	refBytes := make([]byte, 16)
	rand.Read(refBytes)
	reference := hex.EncodeToString(refBytes)

	// Save a secret
	err = vault.Save(ctx, reference, "test-value", nil)
	require.NoError(t, err)

	// Verify it exists
	_, err = vault.Get(ctx, reference)
	require.NoError(t, err)

	// Delete the secret
	err = vault.Delete(ctx, reference)
	require.NoError(t, err)

	// Verify it's gone
	_, err = vault.Get(ctx, reference)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVaultDeleteNotFound(t *testing.T) {
	tdb := NewTestDB(t)
	defer tdb.Cleanup(t)

	key := make([]byte, 32)
	rand.Read(key)
	vault, err := NewVault(tdb.DB, key)
	require.NoError(t, err)

	ctx := context.Background()
	err = vault.Delete(ctx, "nonexistent-reference")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestVaultEncryptionDifferentKeys(t *testing.T) {
	// Create two vaults with different keys
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	rand.Read(key1)
	rand.Read(key2)

	enc1, err := newVaultEncryption(key1)
	require.NoError(t, err)
	enc2, err := newVaultEncryption(key2)
	require.NoError(t, err)

	plaintext := []byte("secret data")

	// Encrypt with first key
	ciphertext, err := enc1.Encrypt(plaintext)
	require.NoError(t, err)

	// Attempt to decrypt with second key should fail
	_, err = enc2.Decrypt(ciphertext)
	assert.Error(t, err)
}
