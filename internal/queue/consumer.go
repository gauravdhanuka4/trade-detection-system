package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gauravdhanuka4/trade-detection-system/internal/models"
	"github.com/gauravdhanuka4/trade-detection-system/internal/utils"
	"github.com/redis/go-redis/v9"
)

// TradeProcessor defines the interface for processing trades
type TradeProcessor interface {
	Process(ctx context.Context, trade *models.Trade) error
}

// StreamConsumer consumes trades from Redis Streams
type StreamConsumer struct {
	redisClient *redis.Client
	processor   TradeProcessor
	config      models.WorkerConfig
	stopChan    chan struct{}
	doneChan    chan struct{}
}

// NewStreamConsumer creates a new stream consumer
func NewStreamConsumer(
	redisClient *redis.Client,
	processor TradeProcessor,
	config models.WorkerConfig,
) *StreamConsumer {
	return &StreamConsumer{
		redisClient: redisClient,
		processor:   processor,
		config:      config,
		stopChan:    make(chan struct{}),
		doneChan:    make(chan struct{}),
	}
}

// Start begins consuming from the Redis Stream
func (sc *StreamConsumer) Start(ctx context.Context) error {
	utils.Logger.Info("Starting stream consumer",
		"stream", sc.config.StreamName,
		"group", sc.config.ConsumerGroup,
		"consumer", sc.config.ConsumerName,
	)

	// Create consumer group if it doesn't exist
	if err := sc.createConsumerGroup(ctx); err != nil {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Start consuming
	go sc.consume(ctx)

	return nil
}

// Stop gracefully stops the consumer
func (sc *StreamConsumer) Stop(ctx context.Context) error {
	utils.Logger.Info("Stopping stream consumer...")

	// Signal stop
	close(sc.stopChan)

	// Wait for completion or timeout
	select {
	case <-sc.doneChan:
		utils.Logger.Info("Stream consumer stopped gracefully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("stream consumer stop timeout: %w", ctx.Err())
	}
}

// createConsumerGroup creates the consumer group if it doesn't exist
func (sc *StreamConsumer) createConsumerGroup(ctx context.Context) error {
	// Try to create the consumer group
	// If it already exists, Redis will return an error, which we can ignore
	err := sc.redisClient.XGroupCreateMkStream(
		ctx,
		sc.config.StreamName,
		sc.config.ConsumerGroup,
		"0", // Start from the beginning
	).Err()

	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}

	utils.Logger.Info("Consumer group ready",
		"group", sc.config.ConsumerGroup,
		"stream", sc.config.StreamName,
	)

	return nil
}

// consume is the main consumption loop
func (sc *StreamConsumer) consume(ctx context.Context) {
	defer close(sc.doneChan)

	pollInterval := time.Duration(sc.config.PollInterval) * time.Millisecond
	processingTimeout := time.Duration(sc.config.ProcessingTimeout) * time.Second

	utils.Logger.Info("Consumer loop started",
		"poll_interval_ms", sc.config.PollInterval,
		"batch_size", sc.config.BatchSize,
	)

	for {
		select {
		case <-sc.stopChan:
			utils.Logger.Info("Consumer loop stopping...")
			return
		case <-ctx.Done():
			utils.Logger.Info("Consumer context cancelled")
			return
		default:
			// Read from stream
			if err := sc.readAndProcess(ctx, processingTimeout); err != nil {
				utils.Logger.Error("Error reading from stream", "error", err)
				// Sleep before retrying
				time.Sleep(pollInterval)
			}
		}
	}
}

// readAndProcess reads a batch of messages and processes them
func (sc *StreamConsumer) readAndProcess(ctx context.Context, timeout time.Duration) error {
	// Read from consumer group
	streams, err := sc.redisClient.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    sc.config.ConsumerGroup,
		Consumer: sc.config.ConsumerName,
		Streams:  []string{sc.config.StreamName, ">"}, // ">" means only new messages
		Count:    int64(sc.config.BatchSize),
		Block:    time.Duration(sc.config.PollInterval) * time.Millisecond,
	}).Result()

	if err != nil {
		// Check if it's just a timeout (no messages)
		if err.Error() == "redis: nil" {
			return nil // No messages, not an error
		}
		return fmt.Errorf("failed to read from stream: %w", err)
	}

	// Process messages
	for _, stream := range streams {
		for _, message := range stream.Messages {
			if err := sc.processMessage(ctx, message, timeout); err != nil {
				utils.Logger.Error("Failed to process message",
					"message_id", message.ID,
					"error", err,
				)
				// Continue processing other messages
				continue
			}

			// Acknowledge the message
			if err := sc.acknowledgeMessage(ctx, message.ID); err != nil {
				utils.Logger.Error("Failed to acknowledge message",
					"message_id", message.ID,
					"error", err,
				)
			}
		}
	}

	return nil
}

// processMessage processes a single message
func (sc *StreamConsumer) processMessage(ctx context.Context, message redis.XMessage, timeout time.Duration) error {
	// Create context with timeout for processing
	processCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Extract trade data from message
	tradeJSON, ok := message.Values["trade_data"].(string)
	if !ok {
		return fmt.Errorf("message does not contain 'trade_data' field")
	}

	// Parse trade
	var trade models.Trade
	if err := json.Unmarshal([]byte(tradeJSON), &trade); err != nil {
		return fmt.Errorf("failed to unmarshal trade: %w", err)
	}

	// Process the trade
	startTime := time.Now()
	if err := sc.processor.Process(processCtx, &trade); err != nil {
		utils.Logger.Error("Failed to process trade",
			"trade_id", trade.ID,
			"user_id", trade.UserID,
			"error", err,
		)
		return err
	}

	processingTime := time.Since(startTime)
	utils.Logger.Debug("Trade processed",
		"trade_id", trade.ID,
		"user_id", trade.UserID,
		"symbol", trade.Symbol,
		"duration_ms", processingTime.Milliseconds(),
	)

	return nil
}

// acknowledgeMessage acknowledges a processed message
func (sc *StreamConsumer) acknowledgeMessage(ctx context.Context, messageID string) error {
	return sc.redisClient.XAck(
		ctx,
		sc.config.StreamName,
		sc.config.ConsumerGroup,
		messageID,
	).Err()
}
