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

	logger.Log.Info("Starting Connect4 Backend Server",
		zap.String("env", cfg.Server.Env),
		zap.String("port", cfg.Server.Port),
	)

	// Connect to database
	db, err := database.New(cfg)
	if err != nil {
		logger.Log.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	// Initialize services
	gameService := services.NewGameService(db)
	matchmakingService := services.NewMatchmakingService(db, cfg, gameService)
	reconnectionService := services.NewReconnectionService(cfg, gameService)
	leaderboardService := services.NewLeaderboardService(db)

	// Initialize handlers
	wsHandler := handlers.NewWSHandler(matchmakingService, gameService, reconnectionService)
	httpHandler := handlers.NewHTTPHandler(leaderboardService)
	gameHandler := handlers.NewGameHandler(db)

	// Setup Gin
	if cfg.Server.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	
	// Middleware
	r.Use(gin.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.ErrorHandler())

	// Routes
	// WebSocket
	r.GET("/ws", wsHandler.HandleWebSocket)

	// API Routes
	api := r.Group("/api")
	{
		api.GET("/health", gameHandler.GetHealth)
		api.GET("/leaderboard", httpHandler.GetLeaderboard)
		api.GET("/player/:username", httpHandler.GetPlayerStats)
	}

	// Start server
	go func() {
		addr := ":" + cfg.Server.Port
		logger.Log.Info("Server listening", zap.String("address", addr))
		if err := r.Run(addr); err != nil {
			logger.Log.Fatal("Server failed to start", zap.Error(err))
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down server...")
}
