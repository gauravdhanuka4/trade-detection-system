package generator

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/config"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/patterns"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/profiles"
	"github.com/google/uuid"
)

// Generator handles trade feed generation
type Generator struct {
	cfg              *config.Config
	redisClient      redis.RedisClient
	profiles         []profiles.TraderProfile
	patternGenerator *patterns.PatternGenerator
	stats            *Statistics
}

// Statistics tracks generation statistics
type Statistics struct {
	TotalTrades     atomic.Int64
	FraudPatterns   atomic.Int64
	VolumeGenerated atomic.Uint64 // In cents to avoid float precision issues
	ByProfile       map[string]*atomic.Int64
	BySymbol        map[string]*atomic.Int64
	StartTime       time.Time
}

// NewGenerator creates a new trade generator
func NewGenerator(cfg *config.Config, redisClient redis.RedisClient) *Generator {
	return &Generator{
		cfg:              cfg,
		redisClient:      redisClient,
		profiles:         profiles.GetDefaultProfiles(),
		patternGenerator: patterns.NewPatternGenerator(),
		stats: &Statistics{
			ByProfile: make(map[string]*atomic.Int64),
			BySymbol:  make(map[string]*atomic.Int64),
			StartTime: time.Now(),
		},
	}
}

// Run starts the trade generation process
func (g *Generator) Run(ctx context.Context) error {
	fmt.Printf("\n🚀 Starting Trade Feed Generator...\n")
	fmt.Printf("Configuration:\n")
	fmt.Printf("  Redis: %s\n", g.cfg.RedisAddress())
	fmt.Printf("  Stream: trades:stream\n")
	fmt.Printf("  Throughput: %d trades/sec\n", g.cfg.Generate.TPS)
	fmt.Printf("  Duration: %v\n", g.cfg.Generate.Duration)
	fmt.Printf("  Fraud Rate: %.1f%%\n\n", g.cfg.Generate.FraudRate*100)

	// Initialize profile counters
	for _, profile := range g.profiles {
		g.stats.ByProfile[string(profile.Type)] = &atomic.Int64{}
	}

	// Start statistics reporter
	go g.reportStats(ctx)

	// Calculate tick interval for desired TPS
	tickInterval := time.Second / time.Duration(g.cfg.Generate.TPS)
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()

	// Set deadline if duration is specified
	var deadline time.Time
	if g.cfg.Generate.Duration > 0 {
		deadline = time.Now().Add(g.cfg.Generate.Duration)
	}

	// Generation loop
	for {
		select {
		case <-ctx.Done():
			return g.printFinalStats()
		case <-ticker.C:
			// Check deadline
			if !deadline.IsZero() && time.Now().After(deadline) {
				return g.printFinalStats()
			}

			// Generate and publish trade(s)
			if err := g.generateAndPublish(ctx); err != nil {
				fmt.Printf("Error generating trade: %v\n", err)
			}
		}
	}
}

// generateAndPublish generates and publishes a trade or fraud pattern
func (g *Generator) generateAndPublish(ctx context.Context) error {
	// Decide if this should be a fraud pattern
	if rand.Float64() < g.cfg.Generate.FraudRate {
		return g.generateFraudPattern(ctx)
	}

	// Generate normal trade
	return g.generateNormalTrade(ctx)
}

// generateNormalTrade generates a single normal trade
func (g *Generator) generateNormalTrade(ctx context.Context) error {
	// Select profile based on weighted distribution
	profile := profiles.SelectProfile(
		g.profiles,
		g.cfg.Profiles.HFTRatio,
		g.cfg.Profiles.RegularRatio,
		g.cfg.Profiles.CasualRatio,
	)
	if profile == nil {
		return fmt.Errorf("no profile selected")
	}

	// Generate trade
	trade := g.generateTrade(profile, time.Now())

	// Publish to Redis
	if err := g.redisClient.PublishTradeToStream(ctx, trade); err != nil {
		return fmt.Errorf("failed to publish trade: %w", err)
	}

	// Update statistics
	g.updateStats(trade, profile, false)

	// Verbose output
	if g.cfg.Generate.Verbose {
		fmt.Printf("[%s] %s: %s %.2f @ $%.2f (%s)\n",
			trade.Timestamp.Format("15:04:05"),
			trade.UserID,
			trade.Type,
			trade.Amount,
			trade.Price,
			trade.Symbol,
		)
	}

	return nil
}

// generateFraudPattern generates a fraud pattern (one or more trades)
func (g *Generator) generateFraudPattern(ctx context.Context) error {
	// Parse fraud type
	fraudType := profiles.AllFraud
	switch g.cfg.Generate.FraudType {
	case "WASH":
		fraudType = profiles.WashTrade
	case "VELOCITY":
		fraudType = profiles.VelocitySpike
	case "ANOMALY":
		fraudType = profiles.Anomaly
	}

	// Select fraud profile
	profile := profiles.SelectFraudProfile(g.profiles, fraudType)
	if profile == nil {
		// Fall back to normal trade
		return g.generateNormalTrade(ctx)
	}

	var trades []*models.Trade
	baseTime := time.Now()

	// Generate fraud pattern
	switch profile.FraudPattern {
	case profiles.WashTrade:
		trades = g.patternGenerator.InjectWashTrade(profile, baseTime)
	case profiles.VelocitySpike:
		trades = g.patternGenerator.InjectVelocitySpike(profile, baseTime)
	case profiles.Anomaly:
		trade := g.patternGenerator.InjectAnomaly(profile, baseTime)
		trades = []*models.Trade{trade}
	default:
		return g.generateNormalTrade(ctx)
	}

	// Publish all trades
	for _, trade := range trades {
		if err := g.redisClient.PublishTradeToStream(ctx, trade); err != nil {
			return fmt.Errorf("failed to publish fraud trade: %w", err)
		}
		g.updateStats(trade, profile, true)

		if g.cfg.Generate.Verbose {
			fmt.Printf("[%s] 🚨 FRAUD %s: %s %.2f @ $%.2f (%s)\n",
				trade.Timestamp.Format("15:04:05"),
				profile.FraudPattern,
				trade.Type,
				trade.Amount,
				trade.Price,
				trade.Symbol,
			)
		}
	}

	return nil
}

// generateTrade creates a trade from a profile
func (g *Generator) generateTrade(profile *profiles.TraderProfile, timestamp time.Time) *models.Trade {
	symbol := profile.GetRandomSymbol()
	amount := g.patternGenerator.GenerateAmount(profile)
	price := g.patternGenerator.GetPrice(symbol)

	return &models.Trade{
		ID:        uuid.New(),
		UserID:    profile.UserID,
		Symbol:    symbol,
		Amount:    amount,
		Price:     price,
		Type:      g.patternGenerator.RandomTradeType(),
		Timestamp: timestamp,
	}
}

// updateStats updates generation statistics
func (g *Generator) updateStats(trade *models.Trade, profile *profiles.TraderProfile, isFraud bool) {
	g.stats.TotalTrades.Add(1)

	if isFraud {
		g.stats.FraudPatterns.Add(1)
	}

	// Volume in cents
	volumeCents := uint64(trade.Amount * trade.Price * 100)
	g.stats.VolumeGenerated.Add(volumeCents)

	// Profile stats
	profileType := string(profile.Type)
	if counter, exists := g.stats.ByProfile[profileType]; exists {
		counter.Add(1)
	}

	// Symbol stats
	if _, exists := g.stats.BySymbol[trade.Symbol]; !exists {
		g.stats.BySymbol[trade.Symbol] = &atomic.Int64{}
	}
	g.stats.BySymbol[trade.Symbol].Add(1)
}

// reportStats periodically reports statistics
func (g *Generator) reportStats(ctx context.Context) {
	ticker := time.NewTicker(g.cfg.Generate.StatsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(g.stats.StartTime)
			totalTrades := g.stats.TotalTrades.Load()
			fraudTrades := g.stats.FraudPatterns.Load()
			volumeCents := g.stats.VolumeGenerated.Load()
			volume := float64(volumeCents) / 100.0

			tps := float64(totalTrades) / elapsed.Seconds()

			fmt.Printf("[%s] %d trades | %d fraud | %.1f tps | $%.1fM volume\n",
				formatDuration(elapsed),
				totalTrades,
				fraudTrades,
				tps,
				volume/1000000.0,
			)
		}
	}
}

// printFinalStats prints final generation statistics
func (g *Generator) printFinalStats() error {
	elapsed := time.Since(g.stats.StartTime)
	totalTrades := g.stats.TotalTrades.Load()
	fraudTrades := g.stats.FraudPatterns.Load()
	volumeCents := g.stats.VolumeGenerated.Load()
	volume := float64(volumeCents) / 100.0

	tps := float64(totalTrades) / elapsed.Seconds()

	fmt.Printf("\n=== Final Statistics ===\n")
	fmt.Printf("Duration:       %v\n", elapsed.Round(time.Second))
	fmt.Printf("Total Trades:   %d\n", totalTrades)
	fmt.Printf("Fraud Patterns: %d (%.1f%%)\n",
		fraudTrades,
		float64(fraudTrades)/float64(totalTrades)*100)
	fmt.Printf("Throughput:     %.1f trades/sec\n", tps)
	fmt.Printf("Total Volume:   $%.2f\n\n", volume)

	fmt.Printf("By Profile Type:\n")
	for profileType, counter := range g.stats.ByProfile {
		count := counter.Load()
		if count > 0 {
			fmt.Printf("  %s: %d (%.1f%%)\n",
				profileType,
				count,
				float64(count)/float64(totalTrades)*100)
		}
	}

	fmt.Printf("\nGeneration complete! ✅\n")
	return nil
}

// formatDuration formats a duration as MM:SS
func formatDuration(d time.Duration) string {
	minutes := int(d.Minutes())
	seconds := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}
