package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/app"
	"github.com/gauravdhanuka4/trade-detection-system/internal/processor"
	"github.com/gauravdhanuka4/trade-detection-system/internal/queue"
	"github.com/joho/godotenv"
)

const (
	WORKER string = "worker"
	API    string = "api"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		// Silently continue if .env not found (use system env vars)
	}

	// Initialize context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize application (handles all shared dependencies)
	application := app.NewInstance(ctx, WORKER)
	defer application.Close()

	application.Logger.Info("Starting Trade Detection Worker",
		"consumer_group", application.Config.Worker.ConsumerGroup,
		"consumer_name", application.Config.Worker.ConsumerName,
		"batch_size", application.Config.Worker.BatchSize,
	)

	// Initialize trade processor
	tradeProcessor := processor.NewTradeProcessor(
		application.RuleEngine,
		application.Redis,
		application.Database.Trades,
		application.AlertService,
		application.Database.Users,
		application.Config.Worker,
	)

	// Initialize stats syncer
	statsSyncer := processor.NewStatsSyncer(
		application.Database,
		application.Redis,
		5*time.Minute, // Sync every 5 minutes
	)

	// Preload active users cache on startup
	application.Logger.Info("Preloading cache for active users...")
	if err := statsSyncer.PreloadActiveUsers(ctx, 7); err != nil {
		application.Logger.Warn("Failed to preload cache, continuing anyway", "error", err)
	} else {
		application.Logger.Info("Cache preload completed successfully")
	}

	// Start stats syncer in background
	go func() {
		application.Logger.Info("Starting stats syncer...")
		statsSyncer.Start(ctx) // Start returns nothing, runs until context cancelled
	}()
	defer statsSyncer.Stop()

	// Initialize stream consumer
	consumer := queue.NewStreamConsumer(
		application.Redis.GetClient(),
		tradeProcessor,
		application.Config.Worker,
	)

	application.Logger.Info("Worker initialization complete, starting trade processing...")

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start consumer in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := consumer.Start(ctx); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case <-sigChan:
		application.Logger.Info("Received shutdown signal, gracefully stopping worker...")
	case err := <-errChan:
		application.Logger.Error("Worker encountered an error", "error", err)
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := consumer.Stop(shutdownCtx); err != nil {
		application.Logger.Error("Error during graceful shutdown", "error", err)
	}

	application.Logger.Info("Trade Detection Worker stopped successfully")
}
