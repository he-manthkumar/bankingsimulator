package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Transfer struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	FromAccount  uuid.UUID `gorm:"type:uuid;not null" json:"from_account"`
	ToAccount    uuid.UUID `gorm:"type:uuid;not null" json:"to_account"`
	Amount       float64   `gorm:"not null" json:"amount"`
	TransferMode string    `gorm:"type:varchar(20);not null" json:"transfer_mode"`
	Status       string    `gorm:"type:varchar(20);not null" json:"status"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
}

func (t *Transfer) BeforeCreate(tx *gorm.DB) error {
	t.ID = uuid.New()
	return nil
}		