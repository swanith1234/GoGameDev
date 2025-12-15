package services

import (
	"connect4/internal/database"
	"connect4/internal/models"
)

type LeaderboardService struct {
	db *database.Database
}

func NewLeaderboardService(db *database.Database) *LeaderboardService {
	return &LeaderboardService{db: db}
}

func (ls *LeaderboardService) GetLeaderboard(limit int) ([]models.LeaderboardEntry, error) {
	return ls.db.GetLeaderboard(limit)
}

func (ls *LeaderboardService) GetPlayerStats(username string) (*models.Player, error) {
	return ls.db.GetPlayerByUsername(username)
}
