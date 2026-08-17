package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Trade represents a financial trade in the system
type Trade struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	UserID    string          `json:"user_id" db:"user_id"`
	Symbol    string          `json:"symbol" db:"symbol"`
	Amount    float64         `json:"amount" db:"amount"`
	Price     float64         `json:"price" db:"price"`
	Type      TradeType       `json:"type" db:"trade_type"`
	Timestamp time.Time       `json:"timestamp" db:"timestamp"`
	Source    *string         `json:"source" db:"source"`
	RawData   json.RawMessage `json:"raw_data" db:"raw_data"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}
