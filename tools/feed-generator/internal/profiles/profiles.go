package profiles

import (
	"math/rand"
	"time"
)

// TraderType represents the type of trader
type TraderType string

const (
	HFTTrader     TraderType = "HFT"
	RegularTrader TraderType = "REGULAR"
	CasualTrader  TraderType = "CASUAL"
	FraudTrader   TraderType = "FRAUD"
)

// FraudType represents the type of fraud pattern
type FraudType string

const (
	NoFraud       FraudType = "NONE"
	WashTrade     FraudType = "WASH"
	VelocitySpike FraudType = "VELOCITY"
	Anomaly       FraudType = "ANOMALY"
	AllFraud      FraudType = "ALL"
)

// TraderProfile defines a trader's behavioral characteristics
type TraderProfile struct {
	UserID         string
	Type           TraderType
	TypicalSymbols []string
	AvgTradeSize   float64
	Volatility     float64 // Standard deviation multiplier (0.0-1.0)
	ActiveHours    []int   // Hours when trader is active (0-23)
	TradesPerHour  int     // Expected trades per hour
	FraudPattern   FraudType
}

// Symbol lists for different trader types
var (
	BlueChipSymbols = []string{"AAPL", "MSFT", "GOOGL", "AMZN", "META", "NVDA", "TSLA"}
	PopularSymbols  = []string{"AAPL", "TSLA", "AMZN", "NVDA", "SPY", "QQQ"}
	ETFSymbols      = []string{"SPY", "QQQ", "VTI", "IWM", "DIA"}
	PennyStocks     = []string{"PENNY_A", "PENNY_B", "PENNY_C", "MICRO_X", "MICRO_Y"}
)

// GetDefaultProfiles returns a set of default trader profiles
func GetDefaultProfiles() []TraderProfile {
	return []TraderProfile{
		// High-Frequency Traders (20% of users, 80% of volume)
		{
			UserID:         "HFT_001",
			Type:           HFTTrader,
			TypicalSymbols: BlueChipSymbols,
			AvgTradeSize:   75000,
			Volatility:     0.2,
			ActiveHours:    []int{9, 10, 11, 12, 13, 14, 15},
			TradesPerHour:  100,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "HFT_002",
			Type:           HFTTrader,
			TypicalSymbols: []string{"TSLA", "NVDA", "META", "AMZN"},
			AvgTradeSize:   100000,
			Volatility:     0.3,
			ActiveHours:    []int{9, 10, 11, 12, 13, 14, 15, 16},
			TradesPerHour:  150,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "HFT_003",
			Type:           HFTTrader,
			TypicalSymbols: BlueChipSymbols,
			AvgTradeSize:   50000,
			Volatility:     0.2,
			ActiveHours:    []int{9, 10, 11, 12, 13, 14, 15},
			TradesPerHour:  80,
			FraudPattern:   NoFraud,
		},

		// Regular Traders (70% of users, 18% of volume)
		{
			UserID:         "USER_001",
			Type:           RegularTrader,
			TypicalSymbols: PopularSymbols[:4],
			AvgTradeSize:   5000,
			Volatility:     0.5,
			ActiveHours:    []int{10, 14},
			TradesPerHour:  2,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_002",
			Type:           RegularTrader,
			TypicalSymbols: []string{"AAPL", "MSFT", "GOOGL"},
			AvgTradeSize:   7500,
			Volatility:     0.4,
			ActiveHours:    []int{9, 12, 15},
			TradesPerHour:  3,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_003",
			Type:           RegularTrader,
			TypicalSymbols: PopularSymbols,
			AvgTradeSize:   4000,
			Volatility:     0.6,
			ActiveHours:    []int{11, 14},
			TradesPerHour:  1,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_004",
			Type:           RegularTrader,
			TypicalSymbols: []string{"TSLA", "NVDA", "AMD"},
			AvgTradeSize:   6000,
			Volatility:     0.5,
			ActiveHours:    []int{10, 13},
			TradesPerHour:  2,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_005",
			Type:           RegularTrader,
			TypicalSymbols: PopularSymbols[:3],
			AvgTradeSize:   5500,
			Volatility:     0.4,
			ActiveHours:    []int{9, 14},
			TradesPerHour:  2,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_006",
			Type:           RegularTrader,
			TypicalSymbols: BlueChipSymbols[:4],
			AvgTradeSize:   8000,
			Volatility:     0.3,
			ActiveHours:    []int{10, 15},
			TradesPerHour:  3,
			FraudPattern:   NoFraud,
		},
		{
			UserID:         "USER_007",
			Type:           RegularTrader,
			TypicalSymbols: PopularSymbols,
			AvgTradeSize:   4500,
			Volatility:     0.5,
			ActiveHours:    []int{11, 14},
			TradesPerHour:  1,
			FraudPattern:   NoFraud,
		},

		// Casual Traders (10% of users, 2% of volume)
		{
			UserID:         "CASUAL_001",
			Type:           CasualTrader,
			TypicalSymbols: ETFSymbols[:2],
			AvgTradeSize:   1000,
			Volatility:     0.3,
			ActiveHours:    []int{10},
			TradesPerHour:  1,
			FraudPattern:   NoFraud,
		},

		// Fraud Traders (for testing detection)
		{
			UserID:         "FRAUD_WASH_001",
			Type:           FraudTrader,
			TypicalSymbols: PennyStocks,
			AvgTradeSize:   10000,
			Volatility:     0.1,
			ActiveHours:    []int{9, 10, 11, 12, 13, 14, 15},
			TradesPerHour:  20,
			FraudPattern:   WashTrade,
		},
		{
			UserID:         "FRAUD_VELOCITY_001",
			Type:           FraudTrader,
			TypicalSymbols: PopularSymbols[:3],
			AvgTradeSize:   5000,
			Volatility:     0.2,
			ActiveHours:    []int{14},
			TradesPerHour:  5,
			FraudPattern:   VelocitySpike,
		},
		{
			UserID:         "FRAUD_ANOMALY_001",
			Type:           FraudTrader,
			TypicalSymbols: BlueChipSymbols[:3],
			AvgTradeSize:   3000,
			Volatility:     0.4,
			ActiveHours:    []int{10, 14},
			TradesPerHour:  2,
			FraudPattern:   Anomaly,
		},
	}
}

// SelectProfile selects a random profile based on weighted distribution
func SelectProfile(profiles []TraderProfile, hftRatio, regularRatio, casualRatio float64) *TraderProfile {
	r := rand.Float64()

	// Separate profiles by type
	var hftProfiles, regularProfiles, casualProfiles, fraudProfiles []TraderProfile
	for i := range profiles {
		switch profiles[i].Type {
		case HFTTrader:
			hftProfiles = append(hftProfiles, profiles[i])
		case RegularTrader:
			regularProfiles = append(regularProfiles, profiles[i])
		case CasualTrader:
			casualProfiles = append(casualProfiles, profiles[i])
		case FraudTrader:
			fraudProfiles = append(fraudProfiles, profiles[i])
		}
	}

	// Select based on ratio
	if r < hftRatio {
		if len(hftProfiles) > 0 {
			profile := hftProfiles[rand.Intn(len(hftProfiles))]
			return &profile
		}
	} else if r < hftRatio+regularRatio {
		if len(regularProfiles) > 0 {
			profile := regularProfiles[rand.Intn(len(regularProfiles))]
			return &profile
		}
	} else {
		if len(casualProfiles) > 0 {
			profile := casualProfiles[rand.Intn(len(casualProfiles))]
			return &profile
		}
	}

	// Fallback
	if len(profiles) > 0 {
		profile := profiles[rand.Intn(len(profiles))]
		return &profile
	}
	return nil
}

// SelectFraudProfile selects a random fraud profile
func SelectFraudProfile(profiles []TraderProfile, fraudType FraudType) *TraderProfile {
	var fraudProfiles []TraderProfile
	for i := range profiles {
		if profiles[i].Type == FraudTrader {
			if fraudType == AllFraud || profiles[i].FraudPattern == fraudType {
				fraudProfiles = append(fraudProfiles, profiles[i])
			}
		}
	}

	if len(fraudProfiles) > 0 {
		profile := fraudProfiles[rand.Intn(len(fraudProfiles))]
		return &profile
	}
	return nil
}

// IsActiveNow checks if the trader is active at the current hour
func (p *TraderProfile) IsActiveNow() bool {
	currentHour := time.Now().Hour()
	for _, hour := range p.ActiveHours {
		if hour == currentHour {
			return true
		}
	}
	return false
}

// GetRandomSymbol returns a random symbol from the trader's typical symbols
func (p *TraderProfile) GetRandomSymbol() string {
	if len(p.TypicalSymbols) == 0 {
		return "AAPL"
	}
	// 80% of the time, use typical symbols
	if rand.Float64() < 0.8 {
		return p.TypicalSymbols[rand.Intn(len(p.TypicalSymbols))]
	}
	// 20% exploration of other symbols
	allSymbols := append(append(append([]string{}, BlueChipSymbols...), PopularSymbols...), ETFSymbols...)
	return allSymbols[rand.Intn(len(allSymbols))]
}
