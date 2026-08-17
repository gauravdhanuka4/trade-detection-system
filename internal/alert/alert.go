package alert

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/db"
	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
	"github.com/google/uuid"
)

// Service defines the alert service interface
type Service interface {
	// ProcessAlerts handles the complete alert processing pipeline
	ProcessAlerts(ctx context.Context, trade *models.Trade, alerts []*models.Alert) error

	// EnrichAlert adds context and metadata to an alert
	EnrichAlert(ctx context.Context, alert *models.Alert, trade *models.Trade) error

	// NotifyHighSeverity sends notifications for high-severity alerts
	NotifyHighSeverity(ctx context.Context, alert *models.Alert, trade *models.Trade) error

	// GetAlertsByTrade retrieves all alerts for a specific trade
	GetAlertsByTrade(ctx context.Context, tradeID uuid.UUID) ([]*models.Alert, error)

	// GetAlertsBySeverity retrieves alerts by severity
	GetAlertsBySeverity(ctx context.Context, severity models.SeverityType, limit, offset int) ([]*models.Alert, error)
}

// alertService implements the Service interface
type alertService struct {
	alertRepo db.AlertRepository
	tradeRepo db.TradeRepository
}

// NewAlertService creates a new alert service
func NewAlertService(alertRepo db.AlertRepository, tradeRepo db.TradeRepository) Service {
	return &alertService{
		alertRepo: alertRepo,
		tradeRepo: tradeRepo,
	}
}

// ProcessAlerts handles the complete alert processing pipeline
func (s *alertService) ProcessAlerts(ctx context.Context, trade *models.Trade, alerts []*models.Alert) error {
	if len(alerts) == 0 {
		return nil
	}

	utils.Logger.Info("Processing alerts through service",
		"trade_id", trade.ID,
		"alert_count", len(alerts),
	)

	// Step 1: Enrich alerts with context
	for _, alert := range alerts {
		if err := s.EnrichAlert(ctx, alert, trade); err != nil {
			utils.Logger.Warn("Failed to enrich alert",
				"alert_id", alert.ID,
				"error", err,
			)
			// Continue processing even if enrichment fails
		}
	}

	// Step 2: Persist alerts to database
	if err := s.alertRepo.BatchCreate(ctx, alerts); err != nil {
		return fmt.Errorf("failed to persist alerts: %w", err)
	}

	utils.Logger.Debug("Alerts persisted to database",
		"trade_id", trade.ID,
		"alert_count", len(alerts),
	)

	// Step 3: Handle high-severity alerts
	for _, alert := range alerts {
		if s.isHighSeverity(alert) {
			if err := s.NotifyHighSeverity(ctx, alert, trade); err != nil {
				utils.Logger.Error("Failed to notify high severity alert",
					"alert_id", alert.ID,
					"severity", alert.Severity,
					"error", err,
				)
				// Don't fail the entire process if notification fails
			}
		}
	}

	return nil
}

// EnrichAlert adds context and metadata to an alert
func (s *alertService) EnrichAlert(ctx context.Context, alert *models.Alert, trade *models.Trade) error {
	// Ensure alert has required fields
	if alert.ID == uuid.Nil {
		alert.ID = uuid.New()
	}

	if alert.TradeID == uuid.Nil {
		alert.TradeID = trade.ID
	}

	if alert.Status == "" {
		alert.Status = models.AlertStatusOpen
	}

	now := time.Now()
	if alert.CreatedAt.IsZero() {
		alert.CreatedAt = now
	}
	if alert.UpdatedAt.IsZero() {
		alert.UpdatedAt = now
	}

	// Enrich description with trade context if description exists
	if alert.Description != nil && *alert.Description != "" {
		enrichedDesc := fmt.Sprintf("%s [Trade: %s, User: %s, Symbol: %s, Amount: %.2f, Price: %.2f]",
			*alert.Description,
			trade.ID,
			trade.UserID,
			trade.Symbol,
			trade.Amount,
			trade.Price,
		)
		alert.Description = &enrichedDesc
	}

	return nil
}

// NotifyHighSeverity sends notifications for high-severity alerts
func (s *alertService) NotifyHighSeverity(ctx context.Context, alert *models.Alert, trade *models.Trade) error {
	// Log high-severity alerts
	utils.Logger.Warn("High severity alert detected",
		"alert_id", alert.ID,
		"trade_id", trade.ID,
		"user_id", trade.UserID,
		"risk_type", alert.RiskType,
		"severity", alert.Severity,
		"description", s.getDescriptionSafe(alert.Description),
	)

	// TODO: Future enhancements:
	// - Send webhook notifications
	// - Send Slack/email alerts
	// - Push to external monitoring systems
	// - Create incident tickets for critical alerts

	return nil
}

// GetAlertsByTrade retrieves all alerts for a specific trade
func (s *alertService) GetAlertsByTrade(ctx context.Context, tradeID uuid.UUID) ([]*models.Alert, error) {
	return s.alertRepo.GetByTradeID(ctx, tradeID)
}

// GetAlertsBySeverity retrieves alerts by severity
func (s *alertService) GetAlertsBySeverity(ctx context.Context, severity models.SeverityType, limit, offset int) ([]*models.Alert, error) {
	return s.alertRepo.GetBySeverity(ctx, severity, limit, offset)
}

// Helper methods

// isHighSeverity checks if an alert requires immediate notification
func (s *alertService) isHighSeverity(alert *models.Alert) bool {
	return alert.Severity == models.SeverityHigh || alert.Severity == models.SeverityCritical
}

// getDescriptionSafe safely extracts description from pointer
func (s *alertService) getDescriptionSafe(desc *string) string {
	if desc == nil {
		return ""
	}
	return *desc
}

// ExtractPatternFlags extracts pattern flags from alerts for Redis caching
func ExtractPatternFlags(alerts []*models.Alert) []string {
	flagsMap := make(map[string]bool)

	for _, alert := range alerts {
		if alert.Description != nil {
			desc := strings.ToLower(*alert.Description)

			// Extract pattern types from description
			if strings.Contains(desc, "wash trade") {
				flagsMap["WASH_TRADER"] = true
			}
			if strings.Contains(desc, "velocity") {
				flagsMap["HIGH_VELOCITY"] = true
			}
			if strings.Contains(desc, "anomal") {
				flagsMap["ANOMALOUS_BEHAVIOR"] = true
			}
		}

		// Extract from risk type
		switch alert.RiskType {
		case models.RiskTypeFraud:
			flagsMap["FRAUD_RISK"] = true
		case models.RiskTypeAML:
			flagsMap["AML_RISK"] = true
		case models.RiskTypeWashTrade:
			flagsMap["WASH_TRADER"] = true
		case models.RiskTypeVelocity:
			flagsMap["HIGH_VELOCITY"] = true
		case models.RiskTypeAnomaly:
			flagsMap["ANOMALOUS_BEHAVIOR"] = true
		}
	}

	// Convert map to slice
	flags := make([]string, 0, len(flagsMap))
	for flag := range flagsMap {
		flags = append(flags, flag)
	}

	return flags
}

// CalculateRiskIncrement calculates total risk score increment from alerts
func CalculateRiskIncrement(alerts []*models.Alert) float64 {
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

	return totalIncrement
}
