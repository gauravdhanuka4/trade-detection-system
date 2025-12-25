package patterns

import (
	"math/rand"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/profiles"
	"github.com/google/uuid"
)

// PatternGenerator handles fraud pattern injection
type PatternGenerator struct {
	symbolPrices map[string]float64
}

// NewPatternGenerator creates a new pattern generator
func NewPatternGenerator() *PatternGenerator {
	return &PatternGenerator{
		symbolPrices: getSymbolPrices(),
	}
}

// InjectWashTrade creates a wash trade pattern (buy followed by sell of same symbol)
func (pg *PatternGenerator) InjectWashTrade(profile *profiles.TraderProfile, baseTime time.Time) []*models.Trade {
	symbol := profile.GetRandomSymbol()
	amount := pg.GenerateAmount(profile)
	price := pg.GetPrice(symbol)

	trades := []*models.Trade{
		{
			ID:        uuid.New(),
			UserID:    profile.UserID,
			Symbol:    symbol,
			Amount:    amount,
			Price:     price,
			Type:      models.TradeTypeBuy,
			Timestamp: baseTime,
		},
		{
			ID:        uuid.New(),
			UserID:    profile.UserID,
			Symbol:    symbol,
			Amount:    amount,
			Price:     price * (1 + (rand.Float64()-0.5)*0.001), // Tiny price difference
			Type:      models.TradeTypeSell,
			Timestamp: baseTime.Add(time.Duration(1+rand.Intn(4)) * time.Second), // 1-4 seconds later
		},
	}

	return trades
}

// InjectVelocitySpike creates a sudden burst of trades
func (pg *PatternGenerator) InjectVelocitySpike(profile *profiles.TraderProfile, baseTime time.Time) []*models.Trade {
	numTrades := 10 + rand.Intn(11) // 10-20 trades
	trades := make([]*models.Trade, numTrades)

	symbol := profile.GetRandomSymbol()
	basePrice := pg.GetPrice(symbol)

	for i := 0; i < numTrades; i++ {
		amount := pg.GenerateAmount(profile)
		// Add small variation to price
		price := basePrice * (1 + (rand.Float64()-0.5)*0.02)

		trades[i] = &models.Trade{
			ID:        uuid.New(),
			UserID:    profile.UserID,
			Symbol:    symbol,
			Amount:    amount,
			Price:     price,
			Type:      pg.RandomTradeType(),
			Timestamp: baseTime.Add(time.Duration(i) * time.Second),
		}
	}

	return trades
}

// InjectAnomaly creates an anomalous trade that deviates from normal pattern
func (pg *PatternGenerator) InjectAnomaly(profile *profiles.TraderProfile, baseTime time.Time) *models.Trade {
	anomalyType := rand.Intn(4)

	trade := &models.Trade{
		ID:        uuid.New(),
		UserID:    profile.UserID,
		Symbol:    profile.GetRandomSymbol(),
		Amount:    pg.GenerateAmount(profile),
		Price:     0,
		Type:      pg.RandomTradeType(),
		Timestamp: baseTime,
	}

	switch anomalyType {
	case 0:
		// Massive size (10x normal)
		trade.Amount = profile.AvgTradeSize * 10
		trade.Price = pg.GetPrice(trade.Symbol)
	case 1:
		// Unusual time (middle of night)
		nightHour := 2 + rand.Intn(4) // 2-5 AM
		trade.Timestamp = time.Date(
			baseTime.Year(), baseTime.Month(), baseTime.Day(),
			nightHour, rand.Intn(60), rand.Intn(60), 0, baseTime.Location(),
		)
		trade.Price = pg.GetPrice(trade.Symbol)
	case 2:
		// Penny stock (unusual symbol for this trader)
		trade.Symbol = profiles.PennyStocks[rand.Intn(len(profiles.PennyStocks))]
		trade.Price = rand.Float64()*5 + 0.5 // $0.50-$5.50
	case 3:
		// Unusual price (way above/below market)
		trade.Price = pg.GetPrice(trade.Symbol) * (1 + (rand.Float64()-0.5)*0.5) // ±25% deviation
	}

	return trade
}

// GenerateAmount generates a trade amount using normal distribution
func (pg *PatternGenerator) GenerateAmount(profile *profiles.TraderProfile) float64 {
	mean := profile.AvgTradeSize
	stdDev := mean * profile.Volatility

	// Use normal distribution
	z := rand.NormFloat64()

	amount := mean + z*stdDev

	// Clamp to reasonable bounds
	minAmount := mean * 0.1
	maxAmount := mean * 3.0

	if amount < minAmount {
		amount = minAmount
	}
	if amount > maxAmount {
		amount = maxAmount
	}

	return amount
}

// GetPrice gets the price for a symbol with small random variation
func (pg *PatternGenerator) GetPrice(symbol string) float64 {
	basePrice, exists := pg.symbolPrices[symbol]
	if !exists {
		basePrice = 100.0 // Default price
	}

	// Add ±1% variation
	variation := (rand.Float64() - 0.5) * 0.02
	return basePrice * (1 + variation)
}

// RandomTradeType returns a random trade type (50/50 buy/sell)
func (pg *PatternGenerator) RandomTradeType() models.TradeType {
	if rand.Float64() < 0.5 {
		return models.TradeTypeBuy
	}
	return models.TradeTypeSell
}

// getSymbolPrices returns a map of realistic symbol prices
func getSymbolPrices() map[string]float64 {
	return map[string]float64{
		// Blue chip stocks
		"AAPL":  175.50,
		"MSFT":  378.25,
		"GOOGL": 140.75,
		"AMZN":  155.35,
		"META":  362.80,
		"NVDA":  495.20,
		"TSLA":  242.80,

		// Popular stocks
		"AMD":  142.30,
		"NFLX": 485.60,
		"DIS":  95.40,

		// ETFs
		"SPY": 475.20,
		"QQQ": 405.80,
		"VTI": 245.30,
		"IWM": 198.50,
		"DIA": 382.40,

		// Penny stocks
		"PENNY_A": 2.50,
		"PENNY_B": 1.80,
		"PENNY_C": 3.20,
		"MICRO_X": 0.85,
		"MICRO_Y": 1.25,
	}
}
