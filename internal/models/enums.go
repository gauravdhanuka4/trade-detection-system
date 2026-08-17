package models

// UserRole represents the role of a user in the system
type UserRole string

const (
	UserRoleAdmin   UserRole = "ADMIN"
	UserRoleAnalyst UserRole = "ANALYST"
	UserRoleViewer  UserRole = "VIEWER"
)

// TradeType represents the type of trade (buy or sell)
type TradeType string

const (
	TradeTypeBuy  TradeType = "BUY"
	TradeTypeSell TradeType = "SELL"
)

// RiskType represents the type of risk detected in an alert
type RiskType string

const (
	RiskTypeFraud     RiskType = "FRAUD"
	RiskTypeAML       RiskType = "AML"
	RiskTypeWashTrade RiskType = "WASH_TRADE"
	RiskTypeVelocity  RiskType = "VELOCITY"
	RiskTypeAnomaly   RiskType = "ANOMALY"
	RiskTypeLayering  RiskType = "LAYERING"
	RiskTypePumpDump  RiskType = "PUMP_DUMP"
)

// SeverityType represents the severity level of an alert
type SeverityType string

const (
	SeverityLow      SeverityType = "LOW"
	SeverityMedium   SeverityType = "MEDIUM"
	SeverityHigh     SeverityType = "HIGH"
	SeverityCritical SeverityType = "CRITICAL"
)

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	AlertStatusOpen          AlertStatus = "OPEN"
	AlertStatusInvestigating AlertStatus = "INVESTIGATING"
	AlertStatusResolved      AlertStatus = "RESOLVED"
	AlertStatusFalsePositive AlertStatus = "FALSE_POSITIVE"
)
