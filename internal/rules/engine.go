package rules

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/db"
	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
	"github.com/google/uuid"
)

// RuleEngine orchestrates fraud detection rules
type RuleEngine struct {
	rules  []Rule
	redis  redis.RedisClient
	logger *slog.Logger
}

// NewRuleEngine creates a new rule engine and loads rules from database
func NewRuleEngine(
	ctx context.Context,
	database *db.DB,
	redisClient redis.RedisClient,
	logger *slog.Logger,
) *RuleEngine {
	engine := &RuleEngine{
		rules:  make([]Rule, 0),
		redis:  redisClient,
		logger: logger,
	}

	// Load rules from database
	enabledRules, err := database.Rules.GetEnabled(ctx)
	if err != nil {
		utils.Fatal(logger, "Failed to load rules from database", "error", err)
	}

	logger.Info("Loading rules from database", "count", len(enabledRules))

	// Register all enabled rules
	registered := 0
	for _, dbRule := range enabledRules {
		if err := engine.registerRuleFromDB(dbRule); err != nil {
			logger.Error("Failed to register rule",
				"error", err,
				"rule_id", dbRule.ID,
				"type", dbRule.Type,
			)
			continue
		}
		registered++
	}

	logger.Info("Rule engine initialized",
		"total_rules", len(enabledRules),
		"registered", registered,
		"enabled", engine.GetEnabledRulesCount(),
	)

	return engine
}

// registerRuleFromDB creates and registers a rule from database model
func (re *RuleEngine) registerRuleFromDB(dbRule *models.Rule) error {
	var rule Rule
	var err error

	switch dbRule.Type {
	case models.RiskTypeWashTrade:
		rule, err = NewWashTradeDetector(dbRule)
	case models.RiskTypeVelocity:
		rule, err = NewVelocityDetector(dbRule)
	case models.RiskTypeAnomaly:
		rule, err = NewAnomalyDetector(dbRule)
	default:
		return fmt.Errorf("unknown rule type: %s", dbRule.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to create detector: %w", err)
	}

	re.RegisterRule(rule)
	re.logger.Info("Registered rule from database",
		"type", dbRule.Type,
		"rule_id", dbRule.ID,
		"name", dbRule.Name,
	)

	return nil
}

// RegisterRule registers a detection rule
func (re *RuleEngine) RegisterRule(rule Rule) {
	re.rules = append(re.rules, rule)
	re.logger.Info("Registered rule",
		"name", rule.Name(),
		"type", rule.Type(),
		"enabled", rule.IsEnabled(),
	)
}

// EvaluateTrade evaluates a trade against all registered rules in parallel using granular cache data
func (re *RuleEngine) EvaluateTrade(
	ctx context.Context,
	trade *models.Trade,
	userProfile *redis.UserProfile,
	realtimeMetrics *redis.RealtimeMetrics,
	recentTrades []*models.Trade,
) ([]*models.Alert, error) {
	// Count enabled rules
	enabledRules := make([]Rule, 0)
	for _, rule := range re.rules {
		if rule.IsEnabled() {
			enabledRules = append(enabledRules, rule)
		}
	}

	if len(enabledRules) == 0 {
		return []*models.Alert{}, nil
	}

	// Channel to collect results
	type ruleResult struct {
		alert *models.Alert
		rule  Rule
		err   error
	}

	resultChan := make(chan ruleResult, len(enabledRules))

	// Evaluate all rules in parallel
	for _, rule := range enabledRules {
		go func(r Rule) {
			// Create timeout context for this rule (5 seconds)
			ruleCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			defer cancel()

			// Evaluate rule with granular data
			alert, err := r.Evaluate(ruleCtx, trade, userProfile, realtimeMetrics, recentTrades)

			resultChan <- ruleResult{
				alert: alert,
				rule:  r,
				err:   err,
			}
		}(rule)
	}

	// Collect results
	alerts := make([]*models.Alert, 0)
	for i := 0; i < len(enabledRules); i++ {
		result := <-resultChan

		if result.err != nil {
			// Check if it's a timeout error
			if result.err == context.DeadlineExceeded {
				re.logger.Warn("Rule evaluation timeout",
					"rule", result.rule.Name(),
					"trade_id", trade.ID,
				)
			} else {
				re.logger.Error("Rule evaluation failed",
					"rule", result.rule.Name(),
					"trade_id", trade.ID,
					"error", result.err,
				)
			}
			continue
		}

		// If alert was generated, enrich and add to results
		if result.alert != nil {
			result.alert.ID = uuid.New()
			result.alert.TradeID = trade.ID
			result.alert.Status = models.AlertStatusOpen
			result.alert.CreatedAt = time.Now()
			result.alert.UpdatedAt = time.Now()

			alerts = append(alerts, result.alert)

			re.logger.Info("Alert generated",
				"alert_id", result.alert.ID,
				"rule", result.rule.Name(),
				"trade_id", trade.ID,
				"severity", result.alert.Severity,
				"risk_type", result.alert.RiskType,
			)
		}
	}

	return alerts, nil
}

// GetRegisteredRules returns all registered rules
func (re *RuleEngine) GetRegisteredRules() []Rule {
	return re.rules
}

// GetEnabledRulesCount returns the count of enabled rules
func (re *RuleEngine) GetEnabledRulesCount() int {
	count := 0
	for _, rule := range re.rules {
		if rule.IsEnabled() {
			count++
		}
	}
	return count
}
