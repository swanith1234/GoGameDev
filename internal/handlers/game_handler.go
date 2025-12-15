package handlers

import (
	"connect4/internal/database"
	"net/http"

	"github.com/gin-gonic/gin"
)

type GameHandler struct {
	db *database.Database
}

func NewGameHandler(db *database.Database) *GameHandler {
	return &GameHandler{db: db}
}

func (gh *GameHandler) GetLeaderboard(c *gin.Context) {
	leaderboard, err := gh.db.GetLeaderboard(100)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get leaderboard"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"leaderboard": leaderboard})
}

func (gh *GameHandler) GetHealth(c *gin.Context) {
	if err := gh.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy", "database": "disconnected"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok", "database": "connected"})
}
