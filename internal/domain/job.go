package domain

import (
	"time"
)

// Job represents a task to be executed by an agent
type Job struct {
	BaseEntity
	Type         JobType    `gorm:"type:varchar(50);not null"`
	State        JobState   `gorm:"type:varchar(20);not null"`
	AgentID      UUID       `gorm:"type:uuid;not null"`
	ServiceID    UUID       `gorm:"type:uuid;not null"`
	Priority     int        `gorm:"not null;default:1"`
	RequestData  JSON       `gorm:"type:jsonb"`
	ResultData   JSON       `gorm:"type:jsonb"`
	ErrorMessage string     `gorm:"type:text"`
	ClaimedAt    *time.Time `gorm:""`
	CompletedAt  *time.Time `gorm:""`

	// Relationships
	Agent   *Agent   `gorm:"foreignKey:AgentID"`
	Service *Service `gorm:"foreignKey:ServiceID"`
}

// TableName returns the table name for the job
func (Job) TableName() string {
	return "jobs"
}
