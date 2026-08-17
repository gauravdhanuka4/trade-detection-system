package models

import (
	"encoding/json"
	"time"
)

// TradingUserStats represents aggregated trading statistics for a user
type TradingUserStats struct {
	UserID            string          `db:"user_id" json:"user_id"`
	TotalTrades       int64           `db:"total_trades" json:"total_trades"`
	TotalAmount       float64         `db:"total_amount" json:"total_amount"`
	AvgTradeSize      float64         `db:"avg_trade_size" json:"avg_trade_size"`
	FirstTradeDate    *time.Time      `db:"first_trade_date" json:"first_trade_date,omitempty"`
	LastTradeDate     *time.Time      `db:"last_trade_date" json:"last_trade_date,omitempty"`
	TradingDays       int             `db:"trading_days" json:"trading_days"`
	RiskScore         float64         `db:"risk_score" json:"risk_score"`
	TypicalSymbols    json.RawMessage `db:"typical_symbols" json:"typical_symbols,omitempty"`
	TradingHours      json.RawMessage `db:"trading_hours" json:"trading_hours,omitempty"`
	LastSyncTimestamp time.Time       `db:"last_sync_timestamp" json:"last_sync_timestamp"`
	CreatedAt         time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at" json:"updated_at"`
}

// SymbolFrequency represents symbol trading frequency
type SymbolFrequency map[string]int

// HourFrequency represents hour-of-day trading frequency
type HourFrequency map[int]int

// GetSymbolFrequency parses typical_symbols JSONB
func (s *TradingUserStats) GetSymbolFrequency() (SymbolFrequency, error) {
	if s.TypicalSymbols == nil {
		return SymbolFrequency{}, nil
	}

	var freq SymbolFrequency
	if err := json.Unmarshal(s.TypicalSymbols, &freq); err != nil {
		return nil, err
	}
	return freq, nil
}

// GetHourFrequency parses trading_hours JSONB
func (s *TradingUserStats) GetHourFrequency() (HourFrequency, error) {
	if s.TradingHours == nil {
		return HourFrequency{}, nil
	}

	// JSONB stores as {"9": 12, "14": 28} with string keys
	var strFreq map[string]int
	if err := json.Unmarshal(s.TradingHours, &strFreq); err != nil {
		return nil, err
	}

	// Convert string keys to int
	freq := make(HourFrequency)
	for hourStr, count := range strFreq {
		var hour int
		if err := json.Unmarshal([]byte(hourStr), &hour); err == nil {
			freq[hour] = count
		}
	}
	return freq, nil
}

// GetTopSymbols returns top N symbols by frequency
func (s *TradingUserStats) GetTopSymbols(n int) []string {
	freq, err := s.GetSymbolFrequency()
	if err != nil || len(freq) == 0 {
		return []string{}
	}

	// Sort by frequency
	type symbolCount struct {
		symbol string
		count  int
	}

	symbols := make([]symbolCount, 0, len(freq))
	for symbol, count := range freq {
		symbols = append(symbols, symbolCount{symbol, count})
	}

	// Simple bubble sort (good enough for small lists)
	for i := 0; i < len(symbols); i++ {
		for j := i + 1; j < len(symbols); j++ {
			if symbols[j].count > symbols[i].count {
				symbols[i], symbols[j] = symbols[j], symbols[i]
			}
		}
	}

	// Take top N
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(symbols); i++ {
		result = append(result, symbols[i].symbol)
	}

	return result
}

// GetActiveHours returns hours with trading activity
func (s *TradingUserStats) GetActiveHours() []int {
	freq, err := s.GetHourFrequency()
	if err != nil || len(freq) == 0 {
		return []int{}
	}

	hours := make([]int, 0, len(freq))
	for hour := range freq {
		hours = append(hours, hour)
	}

	return hours
}
