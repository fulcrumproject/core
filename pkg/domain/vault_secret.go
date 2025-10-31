// VaultSecret entity for encrypted secret storage
// Stores encrypted values with unique references for secure retrieval
package domain

// VaultSecret stores encrypted secrets in the vault
type VaultSecret struct {
	BaseEntity
	Reference      string `gorm:"uniqueIndex;not null" json:"reference"`
	EncryptedValue []byte `gorm:"not null" json:"-"`
}

func (VaultSecret) TableName() string {
	return "vault_secrets"
}

