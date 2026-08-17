package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/google/uuid"
)

// WashTradeConditions defines the conditions for wash trade detection
type WashTradeConditions struct {
	TimeWindowSeconds         int     `json:"time_window_seconds"`
	MinTransactions           int     `json:"min_transactions"`
	PriceVarianceThreshold    float64 `json:"price_variance_threshold"`
	AmountSimilarityThreshold float64 `json:"amount_similarity_threshold"`
}

// WashTradeDetector detects wash trading patterns
type WashTradeDetector struct {
	ruleID     uuid.UUID
	name       string
	enabled    bool
	conditions WashTradeConditions
}

// NewWashTradeDetector creates a new wash trade detector from database rule
func NewWashTradeDetector(rule *models.Rule) (*WashTradeDetector, error) {
	var conditions WashTradeConditions
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse wash trade conditions: %w", err)
	}

	return &WashTradeDetector{
		ruleID:     rule.ID,
		name:       rule.Name,
		enabled:    rule.Enabled,
		conditions: conditions,
	}, nil
}

// Name returns the rule name
func (w *WashTradeDetector) Name() string {
	return w.name
}

// Type returns the risk type
func (w *WashTradeDetector) Type() models.RiskType {
	return models.RiskTypeWashTrade
}

// Severity returns the severity level
func (w *WashTradeDetector) Severity() models.SeverityType {
	return models.SeverityHigh
}

// IsEnabled returns whether the rule is enabled
func (w *WashTradeDetector) IsEnabled() bool {
	return w.enabled
}

// GetRuleID returns the database rule ID
func (w *WashTradeDetector) GetRuleID() uuid.UUID {
	return w.ruleID
}

// Evaluate checks if a trade matches wash trading patterns
func (w *WashTradeDetector) Evaluate(
	ctx context.Context,
	trade *models.Trade,
	userProfile *redis.UserProfile,
	realtimeMetrics *redis.RealtimeMetrics,
	recentTrades []*models.Trade,
) (*models.Alert, error) {
	// Convert []*models.Trade to []models.Trade for existing logic
	recentTradesSlice := make([]models.Trade, len(recentTrades))
	for i, t := range recentTrades {
		if t != nil {
			recentTradesSlice[i] = *t
		}
	}

	// Get recent trades for this user and symbol
	matchingTrades := w.findMatchingTrades(trade, recentTradesSlice)

	// If we found matching opposite trades within the time window
	if len(matchingTrades) >= w.conditions.MinTransactions {
		// Build decision criteria
		timeDiffs := w.getTimeDifferences(matchingTrades, trade)
		priceVars := w.getPriceVariances(matchingTrades, trade)
		amountSims := w.getAmountSimilarities(matchingTrades, trade)

		criteria := map[string]interface{}{
			"rule": "WASH_TRADE",
			"thresholds": map[string]interface{}{
				"time_window_seconds":         w.conditions.TimeWindowSeconds,
				"min_transactions":            w.conditions.MinTransactions,
				"price_variance_threshold":    w.conditions.PriceVarianceThreshold,
				"amount_similarity_threshold": w.conditions.AmountSimilarityThreshold,
			},
			"detected_values": map[string]interface{}{
				"opposite_trades_found":    len(matchingTrades),
				"matching_trade_ids":       w.getTradeIDs(matchingTrades),
				"time_differences_seconds": timeDiffs,
				"price_variances":          priceVars,
				"amount_similarities":      amountSims,
			},
			"triggered_conditions": []string{
				fmt.Sprintf("Found %d opposite trades for symbol %s within %d seconds",
					len(matchingTrades), trade.Symbol, w.conditions.TimeWindowSeconds),
				fmt.Sprintf("All price variances below threshold (%.2f%%)", w.conditions.PriceVarianceThreshold*100),
				fmt.Sprintf("All amount similarities above threshold (%.2f%%)", w.conditions.AmountSimilarityThreshold*100),
			},
		}

		criteriaJSON, _ := json.Marshal(criteria)
		criteriaStr := string(criteriaJSON)

		description := fmt.Sprintf(
			"Potential wash trade detected: %s %s %.2f units of %s. "+
				"Found %d opposite trade(s) for the same symbol within %d seconds. "+
				"Matching trade IDs: %v",
			trade.Type,
			trade.Type,
			trade.Amount,
			trade.Symbol,
			len(matchingTrades),
			w.conditions.TimeWindowSeconds,
			w.getTradeIDs(matchingTrades),
		)

		return &models.Alert{
			RuleID:           &w.ruleID,
			RiskType:         w.Type(),
			Severity:         w.Severity(),
			Description:      &description,
			DecisionCriteria: &criteriaStr,
		}, nil
	}

	return nil, nil
}

// findMatchingTrades finds trades that match wash trading pattern
func (w *WashTradeDetector) findMatchingTrades(
	currentTrade *models.Trade,
	recentTrades []models.Trade,
) []models.Trade {
	matches := make([]models.Trade, 0)
	timeWindow := time.Duration(w.conditions.TimeWindowSeconds) * time.Second

	for _, recentTrade := range recentTrades {
		// Skip the current trade itself
		if recentTrade.ID == currentTrade.ID {
			continue
		}

		// Check if it's the same symbol
		if recentTrade.Symbol != currentTrade.Symbol {
			continue
		}

		// Check if it's the opposite trade type (buy vs sell)
		if recentTrade.Type == currentTrade.Type {
			continue
		}

		// Check if within time window
		timeDiff := currentTrade.Timestamp.Sub(recentTrade.Timestamp)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff > timeWindow {
			continue
		}

		// Check if prices are similar (within tolerance)
		priceDiff := currentTrade.Price - recentTrade.Price
		if priceDiff < 0 {
			priceDiff = -priceDiff
		}

		priceVariance := priceDiff / currentTrade.Price
		if priceVariance > w.conditions.PriceVarianceThreshold {
			continue
		}

		// Check if amounts are similar
		amountDiff := currentTrade.Amount - recentTrade.Amount
		if amountDiff < 0 {
			amountDiff = -amountDiff
		}

		amountSimilarity := 1.0 - (amountDiff / currentTrade.Amount)
		if amountSimilarity < w.conditions.AmountSimilarityThreshold {
			continue
		}

		matches = append(matches, recentTrade)
	}

	return matches
}

// getTradeIDs extracts trade IDs from matching trades
func (w *WashTradeDetector) getTradeIDs(trades []models.Trade) []string {
	ids := make([]string, len(trades))
	for i, trade := range trades {
		ids[i] = trade.ID.String()
	}
	return ids
}

// getTimeDifferences calculates time differences for decision criteria
func (w *WashTradeDetector) getTimeDifferences(trades []models.Trade, currentTrade *models.Trade) []int {
	diffs := make([]int, len(trades))
	for i, trade := range trades {
		diff := currentTrade.Timestamp.Sub(trade.Timestamp)
		if diff < 0 {
			diff = -diff
		}
		diffs[i] = int(diff.Seconds())
	}
	return diffs
}

// getPriceVariances calculates price variances for decision criteria
func (w *WashTradeDetector) getPriceVariances(trades []models.Trade, currentTrade *models.Trade) []float64 {
	variances := make([]float64, len(trades))
	for i, trade := range trades {
		priceDiff := currentTrade.Price - trade.Price
		if priceDiff < 0 {
			priceDiff = -priceDiff
		}
		variances[i] = priceDiff / currentTrade.Price
	}
	return variances
}

// getAmountSimilarities calculates amount similarities for decision criteria
func (w *WashTradeDetector) getAmountSimilarities(trades []models.Trade, currentTrade *models.Trade) []float64 {
	similarities := make([]float64, len(trades))
	for i, trade := range trades {
		amountDiff := currentTrade.Amount - trade.Amount
		if amountDiff < 0 {
			amountDiff = -amountDiff
		}
		similarities[i] = 1.0 - (amountDiff / currentTrade.Amount)
	}
	return similarities
}
