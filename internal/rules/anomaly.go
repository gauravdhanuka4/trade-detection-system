package rules

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/google/uuid"
)

// AnomalyConditions defines the conditions for anomaly detection
type AnomalyConditions struct {
	LookbackDays            int     `json:"lookback_days"`
	SizeDeviationThreshold  float64 `json:"size_deviation_threshold"`
	PriceDeviationThreshold float64 `json:"price_deviation_threshold"`
}

// AnomalyDetector detects anomalous trading patterns
type AnomalyDetector struct {
	ruleID     uuid.UUID
	name       string
	enabled    bool
	conditions AnomalyConditions
}

// NewAnomalyDetector creates a new anomaly detector from database rule
func NewAnomalyDetector(rule *models.Rule) (*AnomalyDetector, error) {
	var conditions AnomalyConditions
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse anomaly conditions: %w", err)
	}

	return &AnomalyDetector{
		ruleID:     rule.ID,
		name:       rule.Name,
		enabled:    rule.Enabled,
		conditions: conditions,
	}, nil
}

// Name returns the rule name
func (a *AnomalyDetector) Name() string {
	return a.name
}

// Type returns the risk type
func (a *AnomalyDetector) Type() models.RiskType {
	return models.RiskTypeAnomaly
}

// Severity returns the severity level
func (a *AnomalyDetector) Severity() models.SeverityType {
	return models.SeverityMedium
}

// IsEnabled returns whether the rule is enabled
func (a *AnomalyDetector) IsEnabled() bool {
	return a.enabled
}

// GetRuleID returns the database rule ID
func (a *AnomalyDetector) GetRuleID() uuid.UUID {
	return a.ruleID
}

// Evaluate checks if a trade shows anomalous behavior
func (a *AnomalyDetector) Evaluate(
	ctx context.Context,
	trade *models.Trade,
	userProfile *redis.UserProfile,
	realtimeMetrics *redis.RealtimeMetrics,
	recentTrades []*models.Trade,
) (*models.Alert, error) {
	if userProfile == nil || realtimeMetrics == nil {
		return nil, nil
	}

	// Convert []*models.Trade to []models.Trade
	recentTradesSlice := make([]models.Trade, len(recentTrades))
	for i, t := range recentTrades {
		if t != nil {
			recentTradesSlice[i] = *t
		}
	}

	// Calculate trade size anomaly (Z-score)
	avgSize := realtimeMetrics.AvgTradeSize
	if avgSize == 0 {
		avgSize = trade.Amount // First trade
	}

	// Calculate standard deviation (simplified - using average as approximation)
	stdDev := avgSize * 0.3 // Assume 30% standard deviation
	if stdDev == 0 {
		stdDev = 1.0
	}

	sizeZScore := math.Abs(trade.Amount-avgSize) / stdDev

	// Check for size anomaly
	sizeAnomaly := sizeZScore > a.conditions.SizeDeviationThreshold

	// Calculate price deviation from recent average
	// Get average price from recent trades
	avgPrice := a.calculateAveragePrice(trade.Symbol, recentTradesSlice)
	if avgPrice == 0 {
		avgPrice = trade.Price // First trade for this symbol
	}

	priceDeviation := math.Abs(trade.Price-avgPrice) / avgPrice

	// Check for price anomaly
	priceAnomaly := priceDeviation > a.conditions.PriceDeviationThreshold

	// Alert if either anomaly is detected
	if sizeAnomaly || priceAnomaly {
		criteria := map[string]interface{}{
			"rule": "ANOMALY",
			"thresholds": map[string]interface{}{
				"lookback_days":             a.conditions.LookbackDays,
				"size_deviation_threshold":  a.conditions.SizeDeviationThreshold,
				"price_deviation_threshold": a.conditions.PriceDeviationThreshold,
			},
			"detected_values": map[string]interface{}{
				"trade_size":          trade.Amount,
				"user_avg_size":       avgSize,
				"standard_deviation":  stdDev,
				"z_score":             sizeZScore,
				"trade_price":         trade.Price,
				"symbol_avg_price":    avgPrice,
				"price_deviation_pct": priceDeviation,
			},
			"triggered_conditions": a.buildTriggeredConditions(
				sizeAnomaly, priceAnomaly, sizeZScore, priceDeviation,
			),
		}

		criteriaJSON, _ := json.Marshal(criteria)
		criteriaStr := string(criteriaJSON)

		description := fmt.Sprintf(
			"Anomalous trading behavior detected for %s: ",
			trade.Symbol,
		)

		if sizeAnomaly {
			description += fmt.Sprintf(
				"Trade size (%.2f) is %.1f standard deviations above user average (%.2f). ",
				trade.Amount,
				sizeZScore,
				avgSize,
			)
		}

		if priceAnomaly {
			description += fmt.Sprintf(
				"Price (%.2f) deviates by %.1f%% from recent average (%.2f). ",
				trade.Price,
				priceDeviation*100,
				avgPrice,
			)
		}

		return &models.Alert{
			RuleID:           &a.ruleID,
			RiskType:         a.Type(),
			Severity:         a.Severity(),
			Description:      &description,
			DecisionCriteria: &criteriaStr,
		}, nil
	}

	return nil, nil
}

// calculateAveragePrice calculates the average price for a symbol from recent trades
func (a *AnomalyDetector) calculateAveragePrice(symbol string, recentTrades []models.Trade) float64 {
	var totalPrice float64
	var count int

	for _, trade := range recentTrades {
		if trade.Symbol == symbol {
			totalPrice += trade.Price
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalPrice / float64(count)
}

// buildTriggeredConditions builds the list of triggered conditions
func (a *AnomalyDetector) buildTriggeredConditions(
	sizeAnomaly, priceAnomaly bool,
	sizeZScore, priceDeviation float64,
) []string {
	conditions := []string{}

	if sizeAnomaly {
		conditions = append(conditions,
			fmt.Sprintf("Trade size Z-score (%.2f) exceeds threshold (%.1f)",
				sizeZScore, a.conditions.SizeDeviationThreshold),
		)
	}

	if priceAnomaly {
		conditions = append(conditions,
			fmt.Sprintf("Price deviation (%.1f%%) exceeds threshold (%.1f%%)",
				priceDeviation*100, a.conditions.PriceDeviationThreshold*100),
		)
	}

	return conditions
}
