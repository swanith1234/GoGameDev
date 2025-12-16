package handlers

import (
	"connect4/internal/services"
	"connect4/internal/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService *services.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *services.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GET /api/analytics/stats
func (ah *AnalyticsHandler) GetStatistics(c *gin.Context) {
	stats, err := ah.analyticsService.GetGameStatistics()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "STATS_ERROR", "Failed to fetch statistics")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, stats)
}

// GET /api/analytics/popular-columns
func (ah *AnalyticsHandler) GetPopularColumns(c *gin.Context) {
	columns, err := ah.analyticsService.GetPopularColumns()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "STATS_ERROR", "Failed to fetch column stats")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"columns": columns,
	})
}

// GET /api/analytics/hourly
func (ah *AnalyticsHandler) GetHourlyStats(c *gin.Context) {
	hourlyData, err := ah.analyticsService.GetHourlyGameCount()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "STATS_ERROR", "Failed to fetch hourly stats")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, gin.H{
		"hourly_data": hourlyData,
	})
}

// GET /api/analytics/player/:username
func (ah *AnalyticsHandler) GetPlayerPerformance(c *gin.Context) {
	username := c.Param("username")
	
	performance, err := ah.analyticsService.GetPlayerPerformance(username)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, "PLAYER_NOT_FOUND", "Player not found")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, performance)
}

// GET /api/analytics/trends
func (ah *AnalyticsHandler) GetTrends(c *gin.Context) {
	trends, err := ah.analyticsService.GetTrendingPatterns()
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "STATS_ERROR", "Failed to fetch trends")
		return
	}

	utils.SuccessResponse(c, http.StatusOK, trends)
}
