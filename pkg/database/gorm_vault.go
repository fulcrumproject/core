// Vault implementation for secure secret storage
package database

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"

	"github.com/fulcrumproject/core/pkg/domain"
	"github.com/fulcrumproject/core/pkg/schema"
	"gorm.io/gorm"
)

// vaultEncryption handles AES-256-GCM encryption/decryption
type vaultEncryption struct {
	key []byte // 32 bytes for AES-256
}

// newVaultEncryption creates a new encryption helper
func newVaultEncryption(key []byte) (*vaultEncryption, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes for AES-256, got %d bytes", len(key))
	}
	return &vaultEncryption{key: key}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM
// Returns: nonce + ciphertext
func (ve *vaultEncryption) Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(ve.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM
// Expects: nonce + ciphertext format
func (ve *vaultEncryption) Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(ve.key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// gormVault implements schema.Vault interface
type gormVault struct {
	db         *gorm.DB
	encryption *vaultEncryption
}

// NewVault creates a new vault instance
func NewVault(db *gorm.DB, encryptionKey []byte) (schema.Vault, error) {
	encryption, err := newVaultEncryption(encryptionKey)
	if err != nil {
		return nil, err
	}

	return &gormVault{
		db:         db,
		encryption: encryption,
	}, nil
}

// Save stores a secret securely in the vault
func (v *gormVault) Save(ctx context.Context, reference string, value any, metadata map[string]any) error {
	// Serialize value to JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to serialize secret: %w", err)
	}

	// Encrypt JSON bytes
	encrypted, err := v.encryption.Encrypt(jsonBytes)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Create secret entity
	secret := &domain.VaultSecret{
		Reference:      reference,
		EncryptedValue: encrypted,
	}

	// Save to database
	if err := v.db.WithContext(ctx).Create(secret).Error; err != nil {
		return fmt.Errorf("failed to save secret: %w", err)
	}

	return nil
}

// Get retrieves and decrypts a secret from the vault
func (v *gormVault) Get(ctx context.Context, reference string) (any, error) {
	// Get secret from database
	var secret domain.VaultSecret
	if err := v.db.WithContext(ctx).Where("reference = ?", reference).First(&secret).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("secret not found: %s", reference)
		}
		return nil, fmt.Errorf("failed to retrieve secret: %w", err)
	}

	// Decrypt to JSON bytes
	jsonBytes, err := v.encryption.Decrypt(secret.EncryptedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Deserialize JSON
	var value any
	if err := json.Unmarshal(jsonBytes, &value); err != nil {
		return nil, fmt.Errorf("failed to deserialize secret: %w", err)
	}

	return value, nil
}

// Delete permanently removes a secret from the vault
func (v *gormVault) Delete(ctx context.Context, reference string) error {
	result := v.db.WithContext(ctx).Where("reference = ?", reference).Delete(&domain.VaultSecret{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete secret: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("secret not found: %s", reference)
	}
	return nil
}
