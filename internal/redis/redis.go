package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"

	"github.com/redis/go-redis/v9"
)

// RedisClient interface defines all Redis operations
type RedisClient interface {
	// Stream Operations
	PublishTradeToStream(ctx context.Context, trade *models.Trade) error
	ConsumeTradeStream(ctx context.Context, consumerGroup, consumerName string, handler TradeHandler) error
	AckMessage(ctx context.Context, stream, group, messageID string) error

	// Hot Data Operations (1-24 hours TTL)
	SetRealtimeMetrics(ctx context.Context, userID string, metrics *RealtimeMetrics) error
	GetRealtimeMetrics(ctx context.Context, userID string) (*RealtimeMetrics, error)
	CacheRecentTrades(ctx context.Context, userID string, trades []models.Trade) error
	GetRecentTrades(ctx context.Context, userID string) ([]models.Trade, error)

	// Warm Data Operations (1-7 days TTL)
	SetUserProfile(ctx context.Context, userID string, profile *UserProfile) error
	GetUserProfile(ctx context.Context, userID string) (*UserProfile, error)
	UpdateRiskScore(ctx context.Context, userID string, score float64) error

	// Granular Cache Helper Methods
	UpdateRealtimeMetrics(ctx context.Context, userID string, trade *models.Trade) error
	AppendRecentTrade(ctx context.Context, userID string, trade *models.Trade) error
	UpdateUserProfileIncremental(ctx context.Context, userID string, trade *models.Trade) error
	AddPatternFlags(ctx context.Context, userID string, flags []string) error
	IncreaseRiskScore(ctx context.Context, userID string, increment float64) error
	DecayRiskScore(ctx context.Context, userID string) error

	// Connection Management
	Ping(ctx context.Context) error
	Close() error

	// GetClient returns the underlying go-redis client for advanced operations
	GetClient() *redis.Client
}

// TradeHandler function type for processing trades from stream
type TradeHandler func(ctx context.Context, trade *models.Trade) error

// redisClient implements RedisClient interface
type redisClient struct {
	client *redis.Client
	config models.RedisConfig
}

// Client is an exported type alias for RedisClient interface
type Client = RedisClient

// Data structures for caching
type RealtimeMetrics struct {
	TradeCount    int       `json:"trade_count"`
	TotalVolume   float64   `json:"total_volume"`
	LastTradeTime time.Time `json:"last_trade_time"`
	AvgTradeSize  float64   `json:"avg_trade_size"`
}

type UserProfile struct {
	RiskScore      float64   `json:"risk_score"`
	TypicalSymbols []string  `json:"typical_symbols"`
	TradingHours   []int     `json:"trading_hours"`
	PatternFlags   []string  `json:"pattern_flags"`
	LastUpdated    time.Time `json:"last_updated"`
}

// Redis key constants
const (
	TradeStreamKey        = "trades:stream"
	RealtimeMetricsPrefix = "metrics:user:"
	UserProfilePrefix     = "profile:user:"
	RecentTradesPrefix    = "trades:recent:"
	TradeContextPrefix    = "context:user:"

	// TTL constants
	HotDataTTL  = 24 * time.Hour     // Hot data: 1-24 hours
	WarmDataTTL = 7 * 24 * time.Hour // Warm data: 1-7 days
)

// NewRedisClient creates a new Redis client
func NewRedisClient(config models.RedisConfig) (RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", config.Host, config.Port),
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &redisClient{
		client: rdb,
		config: config,
	}, nil
}

// Stream Operations

func (r *redisClient) PublishTradeToStream(ctx context.Context, trade *models.Trade) error {
	tradeData, err := json.Marshal(trade)
	if err != nil {
		return fmt.Errorf("failed to marshal trade: %w", err)
	}

	args := &redis.XAddArgs{
		Stream: TradeStreamKey,
		Values: map[string]interface{}{
			"trade_id":   trade.ID.String(),
			"user_id":    trade.UserID,
			"symbol":     trade.Symbol,
			"amount":     trade.Amount,
			"price":      trade.Price,
			"trade_type": string(trade.Type),
			"timestamp":  trade.Timestamp.Unix(),
			"trade_data": string(tradeData),
		},
	}

	_, err = r.client.XAdd(ctx, args).Result()
	if err != nil {
		return fmt.Errorf("failed to add trade to stream: %w", err)
	}

	return nil
}

func (r *redisClient) ConsumeTradeStream(ctx context.Context, consumerGroup, consumerName string, handler TradeHandler) error {
	// Create consumer group if it doesn't exist
	err := r.client.XGroupCreateMkStream(ctx, TradeStreamKey, consumerGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read from stream
			streams, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
				Group:    consumerGroup,
				Consumer: consumerName,
				Streams:  []string{TradeStreamKey, ">"},
				Count:    1,
				Block:    time.Second,
			}).Result()

			if err != nil {
				if err == redis.Nil {
					continue // No messages, continue polling
				}
				return fmt.Errorf("failed to read from stream: %w", err)
			}

			// Process messages
			for _, stream := range streams {
				for _, message := range stream.Messages {
					trade, err := r.parseTradeFromMessage(message)
					if err != nil {
						fmt.Printf("Failed to parse trade from message %s: %v\n", message.ID, err)
						continue
					}

					// Process trade with handler
					if err := handler(ctx, trade); err != nil {
						fmt.Printf("Failed to process trade %s: %v\n", trade.ID, err)
						continue
					}

					// Acknowledge message
					if err := r.AckMessage(ctx, TradeStreamKey, consumerGroup, message.ID); err != nil {
						fmt.Printf("Failed to acknowledge message %s: %v\n", message.ID, err)
					}
				}
			}
		}
	}
}

func (r *redisClient) AckMessage(ctx context.Context, stream, group, messageID string) error {
	return r.client.XAck(ctx, stream, group, messageID).Err()
}

func (r *redisClient) parseTradeFromMessage(message redis.XMessage) (*models.Trade, error) {
	tradeDataStr, ok := message.Values["trade_data"].(string)
	if !ok {
		return nil, fmt.Errorf("trade_data not found in message")
	}

	var trade models.Trade
	if err := json.Unmarshal([]byte(tradeDataStr), &trade); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	return &trade, nil
}

// Hot Data Operations

func (r *redisClient) SetRealtimeMetrics(ctx context.Context, userID string, metrics *RealtimeMetrics) error {
	key := RealtimeMetricsPrefix + userID
	data, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	return r.client.Set(ctx, key, data, HotDataTTL).Err()
}

func (r *redisClient) GetRealtimeMetrics(ctx context.Context, userID string) (*RealtimeMetrics, error) {
	key := RealtimeMetricsPrefix + userID
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	var metrics RealtimeMetrics
	if err := json.Unmarshal([]byte(data), &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
	}

	return &metrics, nil
}

func (r *redisClient) CacheRecentTrades(ctx context.Context, userID string, trades []models.Trade) error {
	key := RecentTradesPrefix + userID
	data, err := json.Marshal(trades)
	if err != nil {
		return fmt.Errorf("failed to marshal trades: %w", err)
	}

	return r.client.Set(ctx, key, data, HotDataTTL).Err()
}

func (r *redisClient) GetRecentTrades(ctx context.Context, userID string) ([]models.Trade, error) {
	key := RecentTradesPrefix + userID
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get recent trades: %w", err)
	}

	var trades []models.Trade
	if err := json.Unmarshal([]byte(data), &trades); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trades: %w", err)
	}

	return trades, nil
}

// Warm Data Operations

func (r *redisClient) SetUserProfile(ctx context.Context, userID string, profile *UserProfile) error {
	key := UserProfilePrefix + userID
	profile.LastUpdated = time.Now()
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("failed to marshal profile: %w", err)
	}

	return r.client.Set(ctx, key, data, WarmDataTTL).Err()
}

func (r *redisClient) GetUserProfile(ctx context.Context, userID string) (*UserProfile, error) {
	key := UserProfilePrefix + userID
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	var profile UserProfile
	if err := json.Unmarshal([]byte(data), &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	return &profile, nil
}

func (r *redisClient) UpdateRiskScore(ctx context.Context, userID string, score float64) error {
	profile, err := r.GetUserProfile(ctx, userID)
	if err != nil {
		return err
	}
	if profile == nil {
		// Create new profile if doesn't exist
		profile = &UserProfile{
			RiskScore: score,
		}
	} else {
		profile.RiskScore = score
	}

	return r.SetUserProfile(ctx, userID, profile)
}

// Helper Methods for Granular Cache Updates

// UpdateRealtimeMetrics updates hot cache metrics incrementally
func (r *redisClient) UpdateRealtimeMetrics(ctx context.Context, userID string, trade *models.Trade) error {
	metrics, err := r.GetRealtimeMetrics(ctx, userID)
	if err != nil || metrics == nil {
		metrics = &RealtimeMetrics{
			TradeCount:   0,
			TotalVolume:  0,
			AvgTradeSize: 0,
		}
	}

	// Incremental update
	metrics.TradeCount++
	metrics.TotalVolume += trade.Amount
	metrics.AvgTradeSize = metrics.TotalVolume / float64(metrics.TradeCount)
	metrics.LastTradeTime = trade.Timestamp

	return r.SetRealtimeMetrics(ctx, userID, metrics)
}

// AppendRecentTrade adds a trade to the recent trades cache
func (r *redisClient) AppendRecentTrade(ctx context.Context, userID string, trade *models.Trade) error {
	trades, err := r.GetRecentTrades(ctx, userID)
	if err != nil || trades == nil {
		trades = []models.Trade{}
	}

	// Append new trade
	trades = append(trades, *trade)

	// Keep only last 50 trades
	if len(trades) > 50 {
		trades = trades[len(trades)-50:]
	}

	return r.CacheRecentTrades(ctx, userID, trades)
}

// UpdateUserProfileIncremental updates profile with new trade data
func (r *redisClient) UpdateUserProfileIncremental(ctx context.Context, userID string, trade *models.Trade) error {
	profile, err := r.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		profile = &UserProfile{
			RiskScore:      0.5, // Neutral for new users
			TypicalSymbols: []string{},
			TradingHours:   []int{},
			PatternFlags:   []string{},
		}
	}

	// Add symbol if not already tracked
	if !contains(profile.TypicalSymbols, trade.Symbol) {
		profile.TypicalSymbols = append(profile.TypicalSymbols, trade.Symbol)
		// Limit to top 10 symbols
		if len(profile.TypicalSymbols) > 10 {
			profile.TypicalSymbols = profile.TypicalSymbols[len(profile.TypicalSymbols)-10:]
		}
	}

	// Add trading hour if not tracked
	hour := trade.Timestamp.Hour()
	if !containsInt(profile.TradingHours, hour) {
		profile.TradingHours = append(profile.TradingHours, hour)
	}

	return r.SetUserProfile(ctx, userID, profile)
}

// AddPatternFlags adds fraud pattern flags to user profile
func (r *redisClient) AddPatternFlags(ctx context.Context, userID string, flags []string) error {
	profile, err := r.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		profile = &UserProfile{
			RiskScore:      0.5,
			TypicalSymbols: []string{},
			TradingHours:   []int{},
			PatternFlags:   []string{},
		}
	}

	// Add unique flags
	for _, flag := range flags {
		if !contains(profile.PatternFlags, flag) {
			profile.PatternFlags = append(profile.PatternFlags, flag)
		}
	}

	return r.SetUserProfile(ctx, userID, profile)
}

// IncreaseRiskScore increases user risk score based on alert severity
func (r *redisClient) IncreaseRiskScore(ctx context.Context, userID string, increment float64) error {
	profile, err := r.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		profile = &UserProfile{
			RiskScore:      0.5,
			TypicalSymbols: []string{},
			TradingHours:   []int{},
			PatternFlags:   []string{},
		}
	}

	// Increase risk score
	profile.RiskScore += increment
	if profile.RiskScore > 1.0 {
		profile.RiskScore = 1.0
	}

	return r.SetUserProfile(ctx, userID, profile)
}

// DecayRiskScore gradually decreases risk score for clean trades
func (r *redisClient) DecayRiskScore(ctx context.Context, userID string) error {
	profile, err := r.GetUserProfile(ctx, userID)
	if err != nil || profile == nil {
		// No profile to decay
		return nil
	}

	// Small decay per clean trade
	profile.RiskScore -= 0.01
	if profile.RiskScore < 0.0 {
		profile.RiskScore = 0.0
	}

	return r.SetUserProfile(ctx, userID, profile)
}

// Connection Management

func (r *redisClient) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *redisClient) Close() error {
	return r.client.Close()
}

func (r *redisClient) GetClient() *redis.Client {
	return r.client
}

// Helper functions

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func containsInt(slice []int, item int) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
