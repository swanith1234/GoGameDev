package main

import (
	"connect4/internal/config"
	"connect4/internal/database"
	"connect4/internal/handlers"
	"connect4/internal/middleware"
	"connect4/internal/services"
	"connect4/pkg/logger"
	
	"fmt"
	"os"
	"os/signal"
	"syscall"


	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		os.Exit(1)
	}

	if err := logger.Init(cfg.Server.Env); err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	logger.Log.Info("🚀 Starting Connect4 Backend Server",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	db, err := database.New(cfg)
	if err != nil {
		logger.Log.Fatal("❌ Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize Kafka Producer
// Initialize Kafka Producer
var kafkaProducer *services.KafkaProducer

if len(cfg.Kafka.Brokers) > 0 {
	var err error
	kafkaProducer, err = services.NewKafkaProducer(cfg)
	if err != nil {
		logger.Log.Warn("⚠️ Kafka producer failed to initialize", zap.Error(err))
		kafkaProducer = nil
	} else {
		defer kafkaProducer.Close()
		logger.Log.Info("✅ Kafka producer initialized successfully")
	}
} else {
	logger.Log.Info("ℹ️ Kafka disabled (no brokers configured)")
}


	// Initialize services
	analyticsService := services.NewAnalyticsService(db)
	gameService := services.NewGameService(db, kafkaProducer)
	matchmakingService := services.NewMatchmakingService(db, cfg, gameService)
	reconnectionService := services.NewReconnectionService(cfg, gameService)
	leaderboardService := services.NewLeaderboardService(db)

	// Initialize handlers
	wsHandler := handlers.NewWSHandler(matchmakingService, gameService, reconnectionService)
	httpHandler := handlers.NewHTTPHandler(leaderboardService)
	gameHandler := handlers.NewGameHandler(db)
	analyticsHandler := handlers.NewAnalyticsHandler(analyticsService)

	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())

	r.GET("/health", gameHandler.GetHealth)
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "Connect 4 Game API",
			"version": "1.0.0",
			"status":  "running",
			"kafka":   kafkaProducer != nil,
		})
	})

	r.GET("/ws", wsHandler.HandleWebSocket)

	api := r.Group("/api")
	{
		api.GET("/health", gameHandler.GetHealth)
		api.GET("/leaderboard", httpHandler.GetLeaderboard)
		api.GET("/player/:username", httpHandler.GetPlayerStats)

		analytics := api.Group("/analytics")
		{
			analytics.GET("/stats", analyticsHandler.GetStatistics)
			analytics.GET("/popular-columns", analyticsHandler.GetPopularColumns)
			analytics.GET("/hourly", analyticsHandler.GetHourlyStats)
			analytics.GET("/player/:username", analyticsHandler.GetPlayerPerformance)
			analytics.GET("/trends", analyticsHandler.GetTrends)
		}
	}

	srv := make(chan error, 1)
	go func() {
		addr := "0.0.0.0:" + cfg.Server.Port
		logger.Log.Info("🌐 Server listening", zap.String("address", addr))
		if err := r.Run(addr); err != nil {
			srv <- err
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		logger.Log.Info("🛑 Shutdown signal received")
	case err := <-srv:
		logger.Log.Fatal("💥 Server failed", zap.Error(err))
	}

	logger.Log.Info("👋 Server stopped")
}
