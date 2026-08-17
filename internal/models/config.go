package models

type ServerConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Database string `mapstructure:"database"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
	SSLMode  string `mapstructure:"ssl_mode"`
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret      string `mapstructure:"secret"`
	ExpiryHours int    `mapstructure:"expiry_hours"`
}

type AuthConfig struct {
	BcryptCost               int  `mapstructure:"bcrypt_cost"`
	RequireEmailVerification bool `mapstructure:"require_email_verification"`
	AllowRegistration        bool `mapstructure:"allow_registration"`
}

type AppConfig struct {
	Environment string `mapstructure:"environment"`
	LogLevel    string `mapstructure:"log_level"`
	Debug       bool   `mapstructure:"debug"`
}

// WorkerConfig holds worker-specific configuration
type WorkerConfig struct {
	ConsumerGroup       string `mapstructure:"consumer_group"`
	ConsumerName        string `mapstructure:"consumer_name"`
	BatchSize           int    `mapstructure:"batch_size"`
	ProcessingTimeout   int    `mapstructure:"processing_timeout"` // seconds
	MaxRetries          int    `mapstructure:"max_retries"`
	StreamName          string `mapstructure:"stream_name"`
	PollInterval        int    `mapstructure:"poll_interval"`         // milliseconds
	SyncIntervalMinutes int    `mapstructure:"sync_interval_minutes"` // Redis DB sync interval
}
