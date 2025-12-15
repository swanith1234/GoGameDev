package handlers

import (
	"connect4/internal/services"
	"connect4/internal/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type HTTPHandler struct {
	leaderboardService *services.LeaderboardService
}

func NewHTTPHandler(leaderboardService *services.LeaderboardService) *HTTPHandler {
	return &HTTPHandler{
		leaderboardService: leaderboardService,
	}
}

func (h *HTTPHandler) GetLeaderboard(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 || limit > 100 {
		limit = 100
	}

	leaderboard, err := h.leaderboardService.GetLeaderboard(limit)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "LEADERBOARD_ERROR", "Failed to fetch leaderboard")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"leaderboard": leaderboard,
		"total":       len(leaderboard),
	})
}

func (h *HTTPHandler) GetPlayerStats(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "INVALID_USERNAME", "Username is required")
		return
	}

	player, err := h.leaderboardService.GetPlayerStats(username)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "PLAYER_ERROR", "Failed to fetch player stats")
		return
	}

	if player == nil {
		utils.ErrorResponse(c, http.StatusNotFound, "PLAYER_NOT_FOUND", "Player not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"player": player,
	})
}
