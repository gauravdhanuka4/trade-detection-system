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

// VelocityConditions defines the conditions for velocity detection
type VelocityConditions struct {
	TimeWindowSeconds  int     `json:"time_window_seconds"`
	MaxTradesPerWindow int     `json:"max_trades_per_window"`
	SpikeMultiplier    float64 `json:"spike_multiplier"`
}

// VelocityDetector detects high-velocity trading patterns
type VelocityDetector struct {
	ruleID     uuid.UUID
	name       string
	enabled    bool
	conditions VelocityConditions
}

// NewVelocityDetector creates a new velocity detector from database rule
func NewVelocityDetector(rule *models.Rule) (*VelocityDetector, error) {
	var conditions VelocityConditions
	if err := json.Unmarshal(rule.Conditions, &conditions); err != nil {
		return nil, fmt.Errorf("failed to parse velocity conditions: %w", err)
	}

	return &VelocityDetector{
		ruleID:     rule.ID,
		name:       rule.Name,
		enabled:    rule.Enabled,
		conditions: conditions,
	}, nil
}

// Name returns the rule name
func (v *VelocityDetector) Name() string {
	return v.name
}

// Type returns the risk type
func (v *VelocityDetector) Type() models.RiskType {
	return models.RiskTypeVelocity
}

// Severity returns the severity level
func (v *VelocityDetector) Severity() models.SeverityType {
	return models.SeverityMedium
}

// IsEnabled returns whether the rule is enabled
func (v *VelocityDetector) IsEnabled() bool {
	return v.enabled
}

// GetRuleID returns the database rule ID
func (v *VelocityDetector) GetRuleID() uuid.UUID {
	return v.ruleID
}

// Evaluate checks if a trade shows high-velocity trading
func (v *VelocityDetector) Evaluate(
	ctx context.Context,
	trade *models.Trade,
	userProfile *redis.UserProfile,
	realtimeMetrics *redis.RealtimeMetrics,
	recentTrades []*models.Trade,
) (*models.Alert, error) {
	if realtimeMetrics == nil {
		return nil, nil
	}

	// Convert []*models.Trade to []models.Trade
	recentTradesSlice := make([]models.Trade, len(recentTrades))
	for i, t := range recentTrades {
		if t != nil {
			recentTradesSlice[i] = *t
		}
	}

	// Count trades in the time window
	timeWindow := time.Duration(v.conditions.TimeWindowSeconds) * time.Second
	tradesInWindow := v.countTradesInWindow(trade, recentTradesSlice, timeWindow)

	// Check if exceeds absolute threshold
	exceededAbsolute := tradesInWindow > v.conditions.MaxTradesPerWindow

	// Calculate user's average velocity for spike detection
	avgVelocity := float64(realtimeMetrics.TradeCount) / 24.0 // trades per hour over 24h
	if avgVelocity < 1.0 {
		avgVelocity = 1.0 // Minimum baseline
	}

	// Calculate current velocity (trades per hour in this window)
	windowHours := float64(v.conditions.TimeWindowSeconds) / 3600.0
	currentVelocity := float64(tradesInWindow) / windowHours

	// Check if it's a spike compared to user's normal behavior
	spikeMultiplier := currentVelocity / avgVelocity
	exceededSpike := spikeMultiplier >= v.conditions.SpikeMultiplier

	// Alert if either condition is met
	if exceededAbsolute || exceededSpike {
		criteria := map[string]interface{}{
			"rule": "VELOCITY",
			"thresholds": map[string]interface{}{
				"time_window_seconds":   v.conditions.TimeWindowSeconds,
				"max_trades_per_window": v.conditions.MaxTradesPerWindow,
				"spike_multiplier":      v.conditions.SpikeMultiplier,
			},
			"detected_values": map[string]interface{}{
				"trades_in_window":           tradesInWindow,
				"time_window_seconds":        v.conditions.TimeWindowSeconds,
				"user_avg_velocity_per_hour": avgVelocity,
				"current_velocity_per_hour":  currentVelocity,
				"spike_multiplier":           spikeMultiplier,
			},
			"triggered_conditions": v.buildTriggeredConditions(
				exceededAbsolute, exceededSpike, tradesInWindow, spikeMultiplier,
			),
		}

		criteriaJSON, _ := json.Marshal(criteria)
		criteriaStr := string(criteriaJSON)

		description := fmt.Sprintf(
			"High velocity trading detected: User executed %d trades within %d seconds. "+
				"Current velocity: %.1f trades/hour (%.1fx their average of %.1f trades/hour). ",
			tradesInWindow,
			v.conditions.TimeWindowSeconds,
			currentVelocity,
			spikeMultiplier,
			avgVelocity,
		)

		if exceededAbsolute {
			description += fmt.Sprintf("Exceeded absolute threshold of %d trades. ", v.conditions.MaxTradesPerWindow)
		}
		if exceededSpike {
			description += fmt.Sprintf("Exceeded spike threshold of %.1fx normal velocity. ", v.conditions.SpikeMultiplier)
		}

		return &models.Alert{
			RuleID:           &v.ruleID,
			RiskType:         v.Type(),
			Severity:         v.Severity(),
			Description:      &description,
			DecisionCriteria: &criteriaStr,
		}, nil
	}

	return nil, nil
}

// countTradesInWindow counts trades within the time window
func (v *VelocityDetector) countTradesInWindow(
	currentTrade *models.Trade,
	recentTrades []models.Trade,
	timeWindow time.Duration,
) int {
	count := 1 // Include current trade

	for _, recentTrade := range recentTrades {
		if recentTrade.ID == currentTrade.ID {
			continue
		}

		timeDiff := currentTrade.Timestamp.Sub(recentTrade.Timestamp)
		if timeDiff < 0 {
			timeDiff = -timeDiff
		}

		if timeDiff <= timeWindow {
			count++
		}
	}

	return count
}

// buildTriggeredConditions builds the list of triggered conditions
func (v *VelocityDetector) buildTriggeredConditions(
	exceededAbsolute, exceededSpike bool,
	tradesInWindow int,
	spikeMultiplier float64,
) []string {
	conditions := []string{}

	if exceededAbsolute {
		conditions = append(conditions,
			fmt.Sprintf("Trade count (%d) exceeds absolute threshold (%d)",
				tradesInWindow, v.conditions.MaxTradesPerWindow),
		)
	}

	if exceededSpike {
		conditions = append(conditions,
			fmt.Sprintf("Spike multiplier (%.2fx) exceeds threshold (%.1fx)",
				spikeMultiplier, v.conditions.SpikeMultiplier),
		)
	}

	return conditions
}
