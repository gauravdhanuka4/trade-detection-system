package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the feed generator
type Config struct {
	Redis    RedisConfig
	Generate GenerateConfig
	Profiles ProfilesConfig
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

// GenerateConfig holds generation settings
type GenerateConfig struct {
	TPS           int
	Duration      time.Duration
	FraudRate     float64
	FraudType     string
	Verbose       bool
	StatsInterval time.Duration
}

// ProfilesConfig holds trader profile distribution settings
type ProfilesConfig struct {
	HFTRatio     float64
	RegularRatio float64
	CasualRatio  float64
}

// LoadConfig loads configuration from Viper
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Redis: RedisConfig{
			Host:     viper.GetString("redis.host"),
			Port:     viper.GetInt("redis.port"),
			Password: viper.GetString("redis.password"),
			DB:       viper.GetInt("redis.db"),
		},
		Generate: GenerateConfig{
			TPS:           viper.GetInt("generate.tps"),
			Duration:      viper.GetDuration("generate.duration"),
			FraudRate:     viper.GetFloat64("generate.fraud_rate"),
			FraudType:     viper.GetString("generate.fraud_type"),
			Verbose:       viper.GetBool("generate.verbose"),
			StatsInterval: viper.GetDuration("generate.stats_interval"),
		},
		Profiles: ProfilesConfig{
			HFTRatio:     viper.GetFloat64("profiles.hft_ratio"),
			RegularRatio: viper.GetFloat64("profiles.regular_ratio"),
			CasualRatio:  viper.GetFloat64("profiles.casual_ratio"),
		},
	}

	// Set defaults if not specified
	if cfg.Redis.Port == 0 {
		cfg.Redis.Port = 6379
	}
	if cfg.Generate.TPS == 0 {
		cfg.Generate.TPS = 100
	}
	if cfg.Generate.Duration == 0 {
		cfg.Generate.Duration = 5 * time.Minute
	}
	if cfg.Generate.StatsInterval == 0 {
		cfg.Generate.StatsInterval = 10 * time.Second
	}
	if cfg.Generate.FraudType == "" {
		cfg.Generate.FraudType = "ALL"
	}
	if cfg.Profiles.HFTRatio == 0 {
		cfg.Profiles.HFTRatio = 0.20
	}
	if cfg.Profiles.RegularRatio == 0 {
		cfg.Profiles.RegularRatio = 0.70
	}
	if cfg.Profiles.CasualRatio == 0 {
		cfg.Profiles.CasualRatio = 0.10
	}

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.Generate.TPS < 1 || c.Generate.TPS > 10000 {
		return fmt.Errorf("tps must be between 1 and 10000, got %d", c.Generate.TPS)
	}
	if c.Generate.FraudRate < 0 || c.Generate.FraudRate > 1 {
		return fmt.Errorf("fraud rate must be between 0.0 and 1.0, got %.2f", c.Generate.FraudRate)
	}

	// Validate profile ratios sum to 1.0
	sum := c.Profiles.HFTRatio + c.Profiles.RegularRatio + c.Profiles.CasualRatio
	if sum < 0.99 || sum > 1.01 {
		return fmt.Errorf("profile ratios must sum to 1.0, got %.2f", sum)
	}

	return nil
}

// RedisAddress returns the full Redis address
func (c *Config) RedisAddress() string {
	return fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.Port)
}
