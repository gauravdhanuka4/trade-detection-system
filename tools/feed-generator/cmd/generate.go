package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/redis"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/config"
	"github.com/gauravdhanuka4/trade-detection-system/tools/feed-generator/internal/generator"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate trade feed",
	Long: `Generate a realistic trade feed with configurable parameters.

The generator creates trades from different trader profiles:
  - High-Frequency Traders (HFT): Fast, high-volume trades
  - Regular Traders: Moderate activity, typical patterns
  - Casual Traders: Low frequency, small volumes

It can inject fraud patterns for testing:
  - Wash Trades: Buy/sell pairs with minimal price difference
  - Velocity Spikes: Sudden bursts of trading activity
  - Anomalies: Unusual patterns (size, time, symbol, price)

Examples:
  # Generate 100 trades per second for 5 minutes
  feed-generator generate --tps 100 --duration 5m

  # Generate with 10% fraud patterns
  feed-generator generate --tps 50 --fraud-rate 0.1

  # Run indefinitely with verbose output
  feed-generator generate --tps 100 --duration 0 --verbose

  # Generate only wash trade patterns
  feed-generator generate --tps 50 --fraud-type WASH`,
	RunE: runGenerate,
}

func init() {
	rootCmd.AddCommand(generateCmd)

	// Local flags
	generateCmd.Flags().IntP("tps", "t", 100,
		"Trades per second (1-10000)")
	generateCmd.Flags().DurationP("duration", "d", 5*time.Minute,
		"Generation duration (0 = infinite)")
	generateCmd.Flags().Float64P("fraud-rate", "f", 0.05,
		"Fraud pattern injection rate (0.0-1.0)")
	generateCmd.Flags().String("fraud-type", "ALL",
		"Fraud types: ALL, WASH, VELOCITY, ANOMALY")
	generateCmd.Flags().BoolP("verbose", "v", false,
		"Print each trade generated")
	generateCmd.Flags().Duration("stats-interval", 10*time.Second,
		"Statistics reporting interval")

	// Bind to viper
	viper.BindPFlag("generate.tps", generateCmd.Flags().Lookup("tps"))
	viper.BindPFlag("generate.duration", generateCmd.Flags().Lookup("duration"))
	viper.BindPFlag("generate.fraud_rate", generateCmd.Flags().Lookup("fraud-rate"))
	viper.BindPFlag("generate.fraud_type", generateCmd.Flags().Lookup("fraud-type"))
	viper.BindPFlag("generate.verbose", generateCmd.Flags().Lookup("verbose"))
	viper.BindPFlag("generate.stats_interval", generateCmd.Flags().Lookup("stats-interval"))
}

func runGenerate(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Connect to Redis
	redisConfig := models.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	}

	redisClient, err := redis.NewRedisClient(redisConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}
	defer redisClient.Close()

	// Test Redis connection
	ctx := context.Background()
	if err := redisClient.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping Redis: %w", err)
	}

	fmt.Printf("✅ Connected to Redis at %s\n", cfg.RedisAddress())

	// Create generator
	gen := generator.NewGenerator(cfg, redisClient)

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Printf("\n\n⚠️  Shutdown signal received, stopping generator...\n")
		cancel()
	}()

	// Run generator
	if err := gen.Run(ctx); err != nil {
		return fmt.Errorf("generator error: %w", err)
	}

	return nil
}
