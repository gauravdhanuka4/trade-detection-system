package config

import (
	"fmt"
	"strings"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   models.ServerConfig   `mapstructure:"server"`
	Database models.DatabaseConfig `mapstructure:"database"`
	Redis    models.RedisConfig    `mapstructure:"redis"`
	JWT      models.JWTConfig      `mapstructure:"jwt"`
	Auth     models.AuthConfig     `mapstructure:"auth"`
	App      models.AppConfig      `mapstructure:"app"`
	Worker   models.WorkerConfig   `mapstructure:"worker"`
}

// Load loads configuration using Viper
func Load() (*Config, error) {
	v := viper.New()

	// Enable reading from environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults first
	setDefaults(v)

	// Try to read from .env file if it exists
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	// Read .env file (optional - will use env vars and defaults if file doesn't exist)
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read .env file: %w", err)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &config, nil
}

// setDefaults sets default values for configuration
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "localhost")
	v.SetDefault("server.port", 8080)

	// Database defaults
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.database", "trade_db")
	v.SetDefault("database.username", "postgres")
	v.SetDefault("database.password", "")
	v.SetDefault("database.ssl_mode", "disable")

	// Redis defaults
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.password", "")
	v.SetDefault("redis.db", 0)

	// JWT defaults
	v.SetDefault("jwt.secret", "your-super-secret-jwt-key-change-in-production")
	v.SetDefault("jwt.expiry_hours", 24)

	// Auth defaults
	v.SetDefault("auth.bcrypt_cost", 12)
	v.SetDefault("auth.require_email_verification", false)
	v.SetDefault("auth.allow_registration", true)

	// App defaults
	v.SetDefault("app.environment", "development")
	v.SetDefault("app.log_level", "info")
	v.SetDefault("app.debug", false)

	// Worker defaults
	v.SetDefault("worker.consumer_group", "trade-processors")
	v.SetDefault("worker.consumer_name", "worker-1")
	v.SetDefault("worker.batch_size", 10)
	v.SetDefault("worker.processing_timeout", 5)
	v.SetDefault("worker.max_retries", 3)
	v.SetDefault("worker.stream_name", "trades:stream")
	v.SetDefault("worker.poll_interval", 1000)
	v.SetDefault("worker.sync_interval_minutes", 5)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Database validation
	if c.Database.Host == "" {
		return fmt.Errorf("database host is required")
	}
	if c.Database.Database == "" {
		return fmt.Errorf("database name is required")
	}
	if c.Database.Username == "" {
		return fmt.Errorf("database username is required")
	}

	// JWT validation
	if c.JWT.Secret == "" || c.JWT.Secret == "your-super-secret-jwt-key-change-in-production" {
		if c.App.Environment == "production" {
			return fmt.Errorf("JWT secret must be set in production")
		}
	}
	if c.JWT.ExpiryHours <= 0 {
		return fmt.Errorf("JWT expiry hours must be positive")
	}

	// Auth validation
	if c.Auth.BcryptCost < 4 || c.Auth.BcryptCost > 31 {
		return fmt.Errorf("bcrypt cost must be between 4 and 31")
	}

	return nil
}

func (c *Config) GetLogLevel() string {
	return c.App.LogLevel
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.App.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.App.Environment == "production"
}
