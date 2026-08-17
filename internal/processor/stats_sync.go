package processor

import (
	"context"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/db"
	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
)

// StatsSyncer handles bidirectional sync between PostgreSQL and Redis
type StatsSyncer struct {
	db          *db.DB
	redisClient redis.RedisClient
	interval    time.Duration
	stopChan    chan struct{}
}

// NewStatsSyncer creates a new stats syncer
func NewStatsSyncer(database *db.DB, redisClient redis.RedisClient, interval time.Duration) *StatsSyncer {
	return &StatsSyncer{
		db:          database,
		redisClient: redisClient,
		interval:    interval,
		stopChan:    make(chan struct{}),
	}
}

// Start begins the periodic sync job
func (s *StatsSyncer) Start(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	utils.Logger.Info("Stats syncer started", "interval", s.interval)

	// Run initial sync
	if err := s.SyncRedisToDB(ctx); err != nil {
		utils.Logger.Error("Initial sync failed", "error", err)
	}

	for {
		select {
		case <-ticker.C:
			if err := s.SyncRedisToDB(ctx); err != nil {
				utils.Logger.Error("Periodic sync failed", "error", err)
			}
		case <-s.stopChan:
			utils.Logger.Info("Stats syncer stopped")
			return
		case <-ctx.Done():
			utils.Logger.Info("Stats syncer context cancelled")
			return
		}
	}
}

// Stop stops the sync job
func (s *StatsSyncer) Stop() {
	close(s.stopChan)
}

// SyncRedisToDB syncs data from Redis to PostgreSQL (periodic sync)
func (s *StatsSyncer) SyncRedisToDB(ctx context.Context) error {
	startTime := time.Now()
	utils.Logger.Info("Starting Redis → PostgreSQL sync")

	// Get all active users from trading_user_stats
	activeUsers, err := s.db.Stats.GetActiveUsers(ctx, 7) // Users active in last 7 days
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	syncCount := 0
	errorCount := 0

	for _, userID := range activeUsers {
		if err := s.syncUserStats(ctx, userID); err != nil {
			utils.Logger.Error("Failed to sync user", "user_id", userID, "error", err)
			errorCount++
			continue
		}
		syncCount++
	}

	duration := time.Since(startTime)
	utils.Logger.Info("Completed Redis → PostgreSQL sync",
		"duration", duration,
		"synced", syncCount,
		"errors", errorCount,
		"total", len(activeUsers))

	return nil
}

// syncUserStats syncs stats for a single user
func (s *StatsSyncer) syncUserStats(ctx context.Context, userID string) error {
	// Get last sync timestamp from DB
	lastSync, err := s.db.Stats.GetLastSyncTime(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get last sync time: %w", err)
	}

	// Count new trades since last sync
	delta, err := s.calculateDelta(ctx, userID, lastSync)
	if err != nil {
		return fmt.Errorf("failed to calculate delta: %w", err)
	}

	// If no new trades, just update risk score from Redis
	if delta.NewCount == 0 {
		return s.updateRiskScoreFromRedis(ctx, userID)
	}

	// Update with delta (ADD, not replace!)
	if err := s.db.Stats.UpdateDelta(ctx, userID, delta.NewCount, delta.NewVolume, delta.LatestTimestamp); err != nil {
		return fmt.Errorf("failed to update delta: %w", err)
	}

	// Also update risk score from Redis profile
	if err := s.updateRiskScoreFromRedis(ctx, userID); err != nil {
		utils.Logger.Warn("Failed to update risk score from Redis", "user_id", userID, "error", err)
	}

	// Update symbols and hours if needed
	if err := s.updatePatternsFromTrades(ctx, userID); err != nil {
		utils.Logger.Warn("Failed to update patterns", "user_id", userID, "error", err)
	}

	return nil
}

// calculateDelta calculates new trades since last sync
func (s *StatsSyncer) calculateDelta(ctx context.Context, userID string, lastSync time.Time) (*tradeDelta, error) {
	// Query trades since last sync
	trades, err := s.db.Trades.GetByUserInTimeWindow(ctx, userID, lastSync, time.Now())
	if err != nil {
		return nil, err
	}

	delta := &tradeDelta{}
	for _, trade := range trades {
		delta.NewCount++
		delta.NewVolume += trade.Amount * trade.Price
		if trade.Timestamp.After(delta.LatestTimestamp) {
			delta.LatestTimestamp = trade.Timestamp
		}
	}

	return delta, nil
}

// updateRiskScoreFromRedis updates risk score from Redis profile
func (s *StatsSyncer) updateRiskScoreFromRedis(ctx context.Context, userID string) error {
	profile, err := s.redisClient.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}

	if profile == nil {
		return nil // No profile in Redis, skip
	}

	return s.db.Stats.UpdateRiskScore(ctx, userID, profile.RiskScore)
}

// updatePatternsFromTrades updates typical_symbols and trading_hours from recent trades
func (s *StatsSyncer) updatePatternsFromTrades(ctx context.Context, userID string) error {
	// Get trades from last 30 days
	trades, err := s.db.Trades.GetByUserInTimeWindow(ctx, userID, time.Now().AddDate(0, 0, -30), time.Now())
	if err != nil {
		return err
	}

	// Calculate symbol frequency
	symbols := make(models.SymbolFrequency)
	hours := make(models.HourFrequency)

	for _, trade := range trades {
		symbols[trade.Symbol]++
		hours[trade.Timestamp.Hour()]++
	}

	return s.db.Stats.UpdateSymbolsAndHours(ctx, userID, symbols, hours)
}

// WarmCacheFromDatabase loads stats from PostgreSQL into Redis (cache warming)
func (s *StatsSyncer) WarmCacheFromDatabase(ctx context.Context, userID string) error {
	utils.Logger.Info("Warming cache from database", "user_id", userID)

	// Get baseline from trading_user_stats
	baseline, err := s.db.Stats.Get(ctx, userID)
	if err != nil {
		// New user, create empty cache
		return s.createEmptyCache(ctx, userID)
	}

	// Get recent trades (last 24 hours)
	recentTrades, err := s.db.Trades.GetRecentTradesByUser(ctx, userID, 24)
	if err != nil {
		utils.Logger.Warn("Failed to get recent trades", "user_id", userID, "error", err)
		recentTrades = []*models.Trade{} // Continue with empty list
	}

	// Calculate hot metrics from recent trades
	metrics := s.calculateMetricsFromTrades(recentTrades)

	// Build warm profile from baseline
	profile := &redis.UserProfile{
		RiskScore:      baseline.RiskScore,
		TypicalSymbols: baseline.GetTopSymbols(10), // Top 10 symbols
		TradingHours:   baseline.GetActiveHours(),
		PatternFlags:   []string{}, // Will be updated as alerts come in
	}

	// Populate Redis
	if err := s.redisClient.SetRealtimeMetrics(ctx, userID, metrics); err != nil {
		return fmt.Errorf("failed to set realtime metrics: %w", err)
	}

	if err := s.redisClient.SetUserProfile(ctx, userID, profile); err != nil {
		return fmt.Errorf("failed to set user profile: %w", err)
	}

	// Store recent trades in Redis
	if len(recentTrades) > 0 {
		// Convert to slice of Trade values (not pointers)
		tradeValues := make([]models.Trade, len(recentTrades))
		for i, t := range recentTrades {
			tradeValues[i] = *t
		}
		if err := s.redisClient.CacheRecentTrades(ctx, userID, tradeValues); err != nil {
			utils.Logger.Warn("Failed to cache recent trades", "user_id", userID, "error", err)
		}
	}

	utils.Logger.Info("Cache warmed successfully", "user_id", userID)
	return nil
}

// createEmptyCache creates an empty cache for a new user
func (s *StatsSyncer) createEmptyCache(ctx context.Context, userID string) error {
	metrics := &redis.RealtimeMetrics{
		TradeCount:    0,
		TotalVolume:   0,
		LastTradeTime: time.Now(),
	}

	profile := &redis.UserProfile{
		RiskScore:      0,
		TypicalSymbols: []string{},
		TradingHours:   []int{},
		PatternFlags:   []string{},
	}

	if err := s.redisClient.SetRealtimeMetrics(ctx, userID, metrics); err != nil {
		return err
	}

	return s.redisClient.SetUserProfile(ctx, userID, profile)
}

// calculateMetricsFromTrades calculates metrics from a list of trades
func (s *StatsSyncer) calculateMetricsFromTrades(trades []*models.Trade) *redis.RealtimeMetrics {
	metrics := &redis.RealtimeMetrics{
		TradeCount:    len(trades),
		TotalVolume:   0,
		LastTradeTime: time.Now(),
	}

	if len(trades) == 0 {
		return metrics
	}

	for _, trade := range trades {
		metrics.TotalVolume += trade.Amount * trade.Price
		if trade.Timestamp.After(metrics.LastTradeTime) {
			metrics.LastTradeTime = trade.Timestamp
		}
	}

	metrics.AvgTradeSize = metrics.TotalVolume / float64(metrics.TradeCount)

	return metrics
}

// tradeDelta represents the change in trading stats
type tradeDelta struct {
	NewCount        int64
	NewVolume       float64
	LatestTimestamp time.Time
}

// PreloadActiveUsers warms cache for all active users (run on startup)
func (s *StatsSyncer) PreloadActiveUsers(ctx context.Context, days int) error {
	utils.Logger.Info("Preloading cache for active users", "days", days)

	activeUsers, err := s.db.Stats.GetActiveUsers(ctx, days)
	if err != nil {
		return fmt.Errorf("failed to get active users: %w", err)
	}

	successCount := 0
	for _, userID := range activeUsers {
		if err := s.WarmCacheFromDatabase(ctx, userID); err != nil {
			utils.Logger.Warn("Failed to warm cache for user", "user_id", userID, "error", err)
			continue
		}
		successCount++
	}

	utils.Logger.Info("Cache preload completed",
		"total", len(activeUsers),
		"success", successCount,
		"failed", len(activeUsers)-successCount)

	return nil
}
