package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/alert"
	"github.com/gauravdhanuka4/trade-detection-system/internal/db"
	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
)

// RuleEngine defines the interface for the rules engine
type RuleEngine interface {
	EvaluateTrade(ctx context.Context, trade *models.Trade, userProfile *redis.UserProfile, realtimeMetrics *redis.RealtimeMetrics, recentTrades []*models.Trade) ([]*models.Alert, error)
}

// TradeProcessor processes incoming trades through fraud detection rules
type TradeProcessor struct {
	ruleEngine   RuleEngine
	redis        redis.RedisClient
	tradeRepo    db.TradeRepository
	alertService alert.Service
	userRepo     db.UserRepository
	config       models.WorkerConfig
}

// NewTradeProcessor creates a new trade processor
func NewTradeProcessor(
	ruleEngine RuleEngine,
	redisClient redis.RedisClient,
	tradeRepo db.TradeRepository,
	alertService alert.Service,
	userRepo db.UserRepository,
	config models.WorkerConfig,
) *TradeProcessor {
	return &TradeProcessor{
		ruleEngine:   ruleEngine,
		redis:        redisClient,
		tradeRepo:    tradeRepo,
		alertService: alertService,
		userRepo:     userRepo,
		config:       config,
	}
}

// Process processes a single trade through the fraud detection pipeline
func (tp *TradeProcessor) Process(ctx context.Context, trade *models.Trade) error {
	startTime := time.Now()

	utils.Logger.Debug("Processing trade",
		"trade_id", trade.ID,
		"user_id", trade.UserID,
		"symbol", trade.Symbol,
		"amount", trade.Amount,
	)

	// Step 1: Persist trade to PostgreSQL (cold storage)
	if err := tp.tradeRepo.Create(ctx, trade); err != nil {
		utils.Logger.Error("Failed to persist trade to database",
			"trade_id", trade.ID,
			"error", err,
		)
		// Don't fail - continue processing for detection
	}

	// Step 2: Update HOT cache (24h TTL) - Fast, every trade
	if err := tp.updateHotCache(ctx, trade); err != nil {
		utils.Logger.Warn("Failed to update hot cache",
			"trade_id", trade.ID,
			"error", err,
		)
	}

	// Step 3: Fetch granular cache data for rule evaluation
	userProfile, _ := tp.redis.GetUserProfile(ctx, trade.UserID)
	realtimeMetrics, _ := tp.redis.GetRealtimeMetrics(ctx, trade.UserID)
	recentTradesSlice, _ := tp.redis.GetRecentTrades(ctx, trade.UserID)

	// Convert []models.Trade to []*models.Trade
	recentTrades := make([]*models.Trade, len(recentTradesSlice))
	for i := range recentTradesSlice {
		recentTrades[i] = &recentTradesSlice[i]
	}

	// Step 4: Run fraud detection rules with granular data
	alerts, err := tp.ruleEngine.EvaluateTrade(ctx, trade, userProfile, realtimeMetrics, recentTrades)
	if err != nil {
		return fmt.Errorf("failed to evaluate trade: %w", err)
	}

	// Step 5: Process generated alerts
	if len(alerts) > 0 {
		if err := tp.processAlerts(ctx, trade, alerts); err != nil {
			utils.Logger.Error("Failed to process alerts",
				"trade_id", trade.ID,
				"alert_count", len(alerts),
				"error", err,
			)
		}

		// Update risk score based on alerts (warm cache)
		if err := tp.updateRiskFromAlerts(ctx, trade.UserID, alerts); err != nil {
			utils.Logger.Warn("Failed to update risk score",
				"user_id", trade.UserID,
				"error", err,
			)
		}

		// Add pattern flags from alerts (warm cache)
		if err := tp.addPatternFlagsFromAlerts(ctx, trade.UserID, alerts); err != nil {
			utils.Logger.Warn("Failed to add pattern flags",
				"user_id", trade.UserID,
				"error", err,
			)
		}
	} else {
		// No alerts - decay risk score slightly
		if err := tp.redis.DecayRiskScore(ctx, trade.UserID); err != nil {
			utils.Logger.Debug("Failed to decay risk score", "error", err)
		}
	}

	// Step 6: Update WARM cache periodically (7d TTL) - Slower, not every trade
	if tp.shouldUpdateProfile(trade) {
		if err := tp.updateWarmCache(ctx, trade); err != nil {
			utils.Logger.Warn("Failed to update warm cache",
				"user_id", trade.UserID,
				"error", err,
			)
		}
	}

	processingTime := time.Since(startTime)
	utils.Logger.Info("Trade processed successfully",
		"trade_id", trade.ID,
		"user_id", trade.UserID,
		"alerts_generated", len(alerts),
		"processing_time_ms", processingTime.Milliseconds(),
	)

	return nil
}

// updateHotCache updates hot cache (24h TTL) - called on EVERY trade
func (tp *TradeProcessor) updateHotCache(ctx context.Context, trade *models.Trade) error {
	// Update realtime metrics incrementally
	if err := tp.redis.UpdateRealtimeMetrics(ctx, trade.UserID, trade); err != nil {
		return fmt.Errorf("failed to update realtime metrics: %w", err)
	}

	// Append to recent trades list
	if err := tp.redis.AppendRecentTrade(ctx, trade.UserID, trade); err != nil {
		return fmt.Errorf("failed to append recent trade: %w", err)
	}

	return nil
}

// updateWarmCache updates warm cache (7d TTL) - called PERIODICALLY
func (tp *TradeProcessor) updateWarmCache(ctx context.Context, trade *models.Trade) error {
	// Update user profile incrementally
	if err := tp.redis.UpdateUserProfileIncremental(ctx, trade.UserID, trade); err != nil {
		return fmt.Errorf("failed to update user profile: %w", err)
	}

	return nil
}

// shouldUpdateProfile determines if warm cache should be updated
func (tp *TradeProcessor) shouldUpdateProfile(trade *models.Trade) bool {
	// Update profile every 10 trades or randomly with 10% probability
	// This reduces write load on warm cache
	return trade.ID.ID()%10 == 0
}

// updateRiskFromAlerts updates risk score based on generated alerts
func (tp *TradeProcessor) updateRiskFromAlerts(ctx context.Context, userID string, alerts []*models.Alert) error {
	totalIncrement := 0.0

	for _, alert := range alerts {
		switch alert.Severity {
		case models.SeverityLow:
			totalIncrement += 0.05
		case models.SeverityMedium:
			totalIncrement += 0.10
		case models.SeverityHigh:
			totalIncrement += 0.20
		case models.SeverityCritical:
			totalIncrement += 0.30
		}
	}

	if totalIncrement > 0 {
		return tp.redis.IncreaseRiskScore(ctx, userID, totalIncrement)
	}

	return nil
}

// addPatternFlagsFromAlerts extracts and adds pattern flags from alerts
func (tp *TradeProcessor) addPatternFlagsFromAlerts(ctx context.Context, userID string, alerts []*models.Alert) error {
	flags := []string{}

	for _, alert := range alerts {
		// Extract pattern type from alert
		if alert.Description != nil {
			desc := *alert.Description
			if contains(desc, "wash trade") || contains(desc, "Wash trade") {
				flags = append(flags, "WASH_TRADER")
			}
			if contains(desc, "velocity") || contains(desc, "Velocity") {
				flags = append(flags, "HIGH_VELOCITY")
			}
			if contains(desc, "anomal") || contains(desc, "Anomal") {
				flags = append(flags, "ANOMALOUS_BEHAVIOR")
			}
		}

		// Also check risk type
		if alert.RiskType == models.RiskTypeFraud {
			flags = append(flags, "FRAUD_RISK")
		}
		if alert.RiskType == models.RiskTypeAML {
			flags = append(flags, "AML_RISK")
		}
	}

	if len(flags) > 0 {
		return tp.redis.AddPatternFlags(ctx, userID, flags)
	}

	return nil
}

// contains checks if a string contains a substring (case-sensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			containsSubstring(s, substr)))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// processAlerts handles generated alerts using the alert service
func (tp *TradeProcessor) processAlerts(ctx context.Context, trade *models.Trade, alerts []*models.Alert) error {
	// Delegate to alert service for complete processing
	return tp.alertService.ProcessAlerts(ctx, trade, alerts)
}
