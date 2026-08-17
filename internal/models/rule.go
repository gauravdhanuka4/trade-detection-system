package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Rule represents a detection rule in the system
type Rule struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	Name       string          `json:"name" db:"name"`
	Type       RiskType        `json:"type" db:"type"`
	Conditions json.RawMessage `json:"conditions" db:"conditions"` // JSONB
	Threshold  *float64        `json:"threshold" db:"threshold"`   // nullable
	Enabled    bool            `json:"enabled" db:"enabled"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}
