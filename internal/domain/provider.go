package domain

// Provider represents a cloud service provider
type Provider struct {
	BaseEntity
	Name        string        `gorm:"not null"`
	State       ProviderState `gorm:"not null"`
	CountryCode CountryCode   `gorm:"size:2"`
	Attributes  Attributes    `gorm:"type:jsonb"`
	Agents      []Agent       `gorm:"foreignKey:ProviderID"`
}

// TableName returns the table name for the provider
func (Provider) TableName() string {
	return "providers"
}

// Validate ensures all Provider fields are valid
func (p Provider) Validate() error {
	if err := p.State.Validate(); err != nil {
		return err
	}

	if err := p.CountryCode.Validate(); err != nil {
		return err
	}

	if p.Attributes != nil {
		return p.Attributes.Validate()
	}
	return nil
}
