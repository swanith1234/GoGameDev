package main

import (
	"connect4/internal/config"
	"connect4/internal/database"
	"connect4/internal/services"
	"connect4/pkg/logger"
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	if err := logger.Init(cfg.Server.Env); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("📊 Starting Connect4 Analytics Consumer",
		zap.String("env", cfg.Server.Env),
		zap.Strings("kafka_brokers", cfg.Kafka.Brokers),
		zap.String("topic", cfg.Kafka.TopicEvents),
	)

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize analytics service
	analyticsService := services.NewAnalyticsService(db)

	// Initialize Kafka consumer
	kafkaConsumer := services.NewKafkaConsumer(cfg, analyticsService)
	defer kafkaConsumer.Close()

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start consumer in goroutine
	go func() {
		logger.Log.Info("🎧 Kafka consumer started, waiting for events...")
		kafkaConsumer.Start(ctx)
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("🛑 Shutdown signal received, stopping consumer...")
	cancel()

	logger.Log.Info("�� Analytics consumer stopped")
}
