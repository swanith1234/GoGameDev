package services

import (
	"connect4/internal/database"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AnalyticsService struct {
	db *database.Database
}

type GameAnalytics struct {
	TotalGames       int     `json:"total_games"`
	GamesToday       int     `json:"games_today"`
	ActiveGames      int     `json:"active_games"`
	AvgGameDuration  float64 `json:"avg_game_duration"`
	BotWinRate       float64 `json:"bot_win_rate"`
	HumanWinRate     float64 `json:"human_win_rate"`
	DrawRate         float64 `json:"draw_rate"`
	TotalPlayers     int     `json:"total_players"`
	ActivePlayers24h int     `json:"active_players_24h"`
	PeakHour         int     `json:"peak_hour"`
	AvgMovesPerGame  float64 `json:"avg_moves_per_game"`
}

type PopularColumn struct {
	Column     int     `json:"column"`
	Count      int     `json:"count"`
	Percentage float64 `json:"percentage"`
}

type HourlyData struct {
	Hour       int `json:"hour"`
	GamesCount int `json:"games_count"`
}

type PlayerPerformance struct {
	Username      string                   `json:"username"`
	GamesPlayed   int                      `json:"games_played"`
	GamesWon      int                      `json:"games_won"`
	WinRate       float64                  `json:"win_rate"`
	AvgGameTime   float64                  `json:"avg_game_time"`
	AvgMoves      float64                  `json:"avg_moves"`
	BotWins       int                      `json:"bot_wins"`
	HumanWins     int                      `json:"human_wins"`
	RecentGames   []map[string]interface{} `json:"recent_games"`
	FavoriteCol   int                      `json:"favorite_column"`
	WinStreak     int                      `json:"win_streak"`
	CurrentStreak int                      `json:"current_streak"`
}

func NewAnalyticsService(db *database.Database) *AnalyticsService {
	return &AnalyticsService{db: db}
}

// Process Kafka Events
func (as *AnalyticsService) ProcessGameStarted(event models.GameStartedEvent) {
	data, _ := json.Marshal(event)
	query := `INSERT INTO game_analytics (game_id, event_type, event_data) VALUES ($1, $2, $3)`
	_, err := as.db.Exec(query, event.GameID, "GAME_STARTED", data)
	if err != nil {
		logger.Log.Error("Failed to store game started event", zap.Error(err))
	}
	logger.Log.Info("Processed GAME_STARTED event", zap.String("game_id", event.GameID.String()))
}

func (as *AnalyticsService) ProcessMoveMade(event models.MoveMadeEvent) {
	data, _ := json.Marshal(event)
	query := `INSERT INTO game_analytics (game_id, event_type, event_data) VALUES ($1, $2, $3)`
	_, err := as.db.Exec(query, event.GameID, "MOVE_MADE", data)
	if err != nil {
		logger.Log.Error("Failed to store move made event", zap.Error(err))
	}
	logger.Log.Debug("Processed MOVE_MADE event", zap.String("game_id", event.GameID.String()))
}

func (as *AnalyticsService) ProcessGameCompleted(event models.GameCompletedEvent) {
	data, _ := json.Marshal(event)
	query := `INSERT INTO game_analytics (game_id, event_type, event_data) VALUES ($1, $2, $3)`
	_, err := as.db.Exec(query, event.GameID, "GAME_COMPLETED", data)
	if err != nil {
		logger.Log.Error("Failed to store game completed event", zap.Error(err))
	}

	// Update metrics
	as.calculateMetrics()
	logger.Log.Info("Processed GAME_COMPLETED event", zap.String("game_id", event.GameID.String()))
}

// Calculate and store aggregated metrics
func (as *AnalyticsService) calculateMetrics() {
	// Average game duration
	var avgDuration float64
	as.db.QueryRow(`SELECT AVG(duration_seconds) FROM games WHERE status = 'completed'`).Scan(&avgDuration)
	as.storeMetric("avg_game_duration", avgDuration, nil)

	// Bot win rate
	var botWins, totalGames int
	as.db.QueryRow(`SELECT COUNT(*) FROM games WHERE player2_is_bot = true AND winner_id = player2_id`).Scan(&botWins)
	as.db.QueryRow(`SELECT COUNT(*) FROM games WHERE player2_is_bot = true AND status = 'completed'`).Scan(&totalGames)
	if totalGames > 0 {
		botWinRate := float64(botWins) / float64(totalGames) * 100
		as.storeMetric("bot_win_rate", botWinRate, nil)
	}

	// Refresh materialized view
	as.db.Exec(`SELECT refresh_analytics_summary()`)
}

func (as *AnalyticsService) storeMetric(name string, value float64, data map[string]interface{}) {
	jsonData, _ := json.Marshal(data)
	query := `INSERT INTO analytics_metrics (metric_name, metric_value, metric_data) VALUES ($1, $2, $3)`
	as.db.Exec(query, name, value, jsonData)
}

// Get overall statistics
func (as *AnalyticsService) GetGameStatistics() (*GameAnalytics, error) {
	stats := &GameAnalytics{}

	// Use prepared query for better performance
	err := as.db.QueryRow(`
		SELECT 
			COUNT(*) as total_games,
			COUNT(*) FILTER (WHERE DATE(started_at) = CURRENT_DATE) as games_today,
			COUNT(*) FILTER (WHERE status = 'active') as active_games,
			COALESCE(AVG(duration_seconds) FILTER (WHERE status = 'completed'), 0) as avg_duration,
			COUNT(DISTINCT player1_id) FILTER (WHERE started_at > NOW() - INTERVAL '24 hours') as active_players,
			COALESCE(AVG(total_moves), 0) as avg_moves
		FROM games
	`).Scan(
		&stats.TotalGames,
		&stats.GamesToday,
		&stats.ActiveGames,
		&stats.AvgGameDuration,
		&stats.ActivePlayers24h,
		&stats.AvgMovesPerGame,
	)

	if err != nil {
		return nil, err
	}

	// Win rates
	var completedGames, botWins, humanWins, draws int
	as.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE status IN ('completed', 'draw', 'forfeited')) as completed,
			COUNT(*) FILTER (WHERE status = 'completed' AND player2_is_bot = true AND winner_id = player2_id) as bot_wins,
			COUNT(*) FILTER (WHERE status = 'completed' AND player2_is_bot = false) as human_wins,
			COUNT(*) FILTER (WHERE status = 'draw') as draws
		FROM games
	`).Scan(&completedGames, &botWins, &humanWins, &draws)

	if completedGames > 0 {
		stats.BotWinRate = float64(botWins) / float64(completedGames) * 100
		stats.HumanWinRate = float64(humanWins) / float64(completedGames) * 100
		stats.DrawRate = float64(draws) / float64(completedGames) * 100
	}

	// Total players
	as.db.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&stats.TotalPlayers)

	// Peak hour
	as.db.QueryRow(`
		SELECT EXTRACT(HOUR FROM started_at) as hour
		FROM games
		WHERE started_at > NOW() - INTERVAL '7 days'
		GROUP BY hour
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`).Scan(&stats.PeakHour)

	return stats, nil
}

// Get popular columns
func (as *AnalyticsService) GetPopularColumns() ([]PopularColumn, error) {
	rows, err := as.db.Query(`
		SELECT 
			column_index, 
			COUNT(*) as count,
			COUNT(*) * 100.0 / (SELECT COUNT(*) FROM game_moves) as percentage
		FROM game_moves
		GROUP BY column_index
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []PopularColumn
	for rows.Next() {
		var col PopularColumn
		if err := rows.Scan(&col.Column, &col.Count, &col.Percentage); err != nil {
			continue
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// Get hourly game distribution
func (as *AnalyticsService) GetHourlyGameCount() ([]HourlyData, error) {
	rows, err := as.db.Query(`
		SELECT 
			EXTRACT(HOUR FROM started_at)::int as hour, 
			COUNT(*)::int as count
		FROM games
		WHERE started_at > NOW() - INTERVAL '24 hours'
		GROUP BY hour
		ORDER BY hour
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var hourlyData []HourlyData
	for rows.Next() {
		var data HourlyData
		if err := rows.Scan(&data.Hour, &data.GamesCount); err != nil {
			continue
		}
		hourlyData = append(hourlyData, data)
	}

	return hourlyData, nil
}

// Get detailed player performance
func (as *AnalyticsService) GetPlayerPerformance(username string) (*PlayerPerformance, error) {
	player, err := as.db.GetPlayerByUsername(username)
	if err != nil || player == nil {
		return nil, err
	}

	perf := &PlayerPerformance{
		Username:    player.Username,
		GamesPlayed: player.GamesPlayed,
		GamesWon:    player.GamesWon,
	}

	if player.GamesPlayed > 0 {
		perf.WinRate = float64(player.GamesWon) / float64(player.GamesPlayed) * 100
	}

	// Average game time and moves
	as.db.QueryRow(`
		SELECT 
			COALESCE(AVG(duration_seconds), 0),
			COALESCE(AVG(total_moves), 0)
		FROM games
		WHERE (player1_id = $1 OR player2_id = $1) AND status = 'completed'
	`, player.ID).Scan(&perf.AvgGameTime, &perf.AvgMoves)

	// Bot vs Human wins
	as.db.QueryRow(`
		SELECT 
			COUNT(*) FILTER (WHERE player2_is_bot = true AND winner_id = $1),
			COUNT(*) FILTER (WHERE player2_is_bot = false AND winner_id = $1)
		FROM games
		WHERE (player1_id = $1 OR player2_id = $1) AND status = 'completed'
	`, player.ID).Scan(&perf.BotWins, &perf.HumanWins)

	// Favorite column
	as.db.QueryRow(`
		SELECT column_index
		FROM game_moves
		WHERE player_id = $1
		GROUP BY column_index
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, player.ID).Scan(&perf.FavoriteCol)

	// Win streak
	perf.WinStreak = as.calculateWinStreak(player.ID)
	perf.CurrentStreak = as.calculateCurrentStreak(player.ID)

	// Recent games
	perf.RecentGames = as.getRecentGames(player.ID, 10)

	return perf, nil
}

func (as *AnalyticsService) calculateWinStreak(playerID int) int {
	rows, _ := as.db.Query(`
		SELECT winner_id = $1 as won
		FROM games
		WHERE (player1_id = $1 OR player2_id = $1) AND status = 'completed'
		ORDER BY completed_at DESC
	`, playerID)
	defer rows.Close()

	maxStreak := 0
	currentStreak := 0

	for rows.Next() {
		var won bool
		rows.Scan(&won)
		if won {
			currentStreak++
			if currentStreak > maxStreak {
				maxStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}
	}

	return maxStreak
}

func (as *AnalyticsService) calculateCurrentStreak(playerID int) int {
	rows, _ := as.db.Query(`
		SELECT winner_id = $1 as won
		FROM games
		WHERE (player1_id = $1 OR player2_id = $1) AND status = 'completed'
		ORDER BY completed_at DESC
	`, playerID)
	defer rows.Close()

	streak := 0
	for rows.Next() {
		var won bool
		rows.Scan(&won)
		if won {
			streak++
		} else {
			break
		}
	}

	return streak
}

func (as *AnalyticsService) getRecentGames(playerID int, limit int) []map[string]interface{} {
	rows, _ := as.db.Query(`
		SELECT 
			g.id,
			g.winner_id = $1 as won,
			g.duration_seconds,
			g.total_moves,
			g.player2_is_bot,
			g.started_at
		FROM games g
		WHERE (g.player1_id = $1 OR g.player2_id = $1)
		AND g.status IN ('completed', 'draw', 'forfeited')
		ORDER BY g.started_at DESC
		LIMIT $2
	`, playerID, limit)
	defer rows.Close()

	games := []map[string]interface{}{}
	for rows.Next() {
		var gameID uuid.UUID
		var won, isBot bool
		var duration, moves int
		var startedAt time.Time

		rows.Scan(&gameID, &won, &duration, &moves, &isBot, &startedAt)
		games = append(games, map[string]interface{}{
			"game_id":    gameID,
			"won":        won,
			"duration":   duration,
			"moves":      moves,
			"vs_bot":     isBot,
			"started_at": startedAt,
		})
	}

	return games
}

// Get trending patterns
func (as *AnalyticsService) GetTrendingPatterns() (map[string]interface{}, error) {
	patterns := make(map[string]interface{})

	// Games per day for last 7 days
	rows, _ := as.db.Query(`
		SELECT DATE(started_at), COUNT(*)
		FROM games
		WHERE started_at > NOW() - INTERVAL '7 days'
		GROUP BY DATE(started_at)
		ORDER BY DATE(started_at)
	`)
	defer rows.Close()

	dailyGames := []map[string]interface{}{}
	for rows.Next() {
		var date time.Time
		var count int
		rows.Scan(&date, &count)
		dailyGames = append(dailyGames, map[string]interface{}{
			"date":  date.Format("2006-01-02"),
			"count": count,
		})
	}
	patterns["daily_games"] = dailyGames

	// Most active players this week
	topPlayers, _ := as.db.Query(`
		SELECT p.username, COUNT(*) as games
		FROM games g
		JOIN players p ON (g.player1_id = p.id OR g.player2_id = p.id)
		WHERE g.started_at > NOW() - INTERVAL '7 days'
		GROUP BY p.username
		ORDER BY games DESC
		LIMIT 5
	`)
	defer topPlayers.Close()

	activeUsers := []map[string]interface{}{}
	for topPlayers.Next() {
		var username string
		var games int
		topPlayers.Scan(&username, &games)
		activeUsers = append(activeUsers, map[string]interface{}{
			"username": username,
			"games":    games,
		})
	}
	patterns["most_active_players"] = activeUsers

	return patterns, nil
}
