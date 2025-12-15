package database

import (
	"connect4/internal/config"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

type Database struct {
	db *sql.DB
}

func New(cfg *config.Config) (*Database, error) {
	dsn := cfg.GetDatabaseDSN()
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Database connected successfully")
	return &Database{db: db}, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) Ping() error {
	return d.db.Ping()
}

func (d *Database) CreatePlayer(username string) (*models.Player, error) {
	var player models.Player
	query := `
		INSERT INTO players (username) 
		VALUES ($1) 
		ON CONFLICT (username) DO UPDATE SET updated_at = CURRENT_TIMESTAMP
		RETURNING id, username, games_played, games_won, created_at, updated_at
	`
	err := d.db.QueryRow(query, username).Scan(
		&player.ID, &player.Username, &player.GamesPlayed,
		&player.GamesWon, &player.CreatedAt, &player.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create player: %w", err)
	}
	return &player, nil
}

func (d *Database) GetPlayerByUsername(username string) (*models.Player, error) {
	var player models.Player
	query := `SELECT id, username, games_played, games_won, created_at, updated_at FROM players WHERE username = $1`
	err := d.db.QueryRow(query, username).Scan(
		&player.ID, &player.Username, &player.GamesPlayed,
		&player.GamesWon, &player.CreatedAt, &player.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get player: %w", err)
	}
	return &player, nil
}

func (d *Database) CreateGame(player1ID int, player2ID *int, isBot bool) (uuid.UUID, error) {
	gameID := uuid.New()
	query := `INSERT INTO games (id, player1_id, player2_id, player2_is_bot, status, started_at) VALUES ($1, $2, $3, $4, $5, $6)`
	_, err := d.db.Exec(query, gameID, player1ID, player2ID, isBot, models.GameStatusActive, time.Now())
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create game: %w", err)
	}
	return gameID, nil
}

func (d *Database) CompleteGame(gameID uuid.UUID, winnerID *int, status models.GameStatus, totalMoves int, startedAt time.Time) error {
	duration := int(time.Since(startedAt).Seconds())
	completedAt := time.Now()
	query := `UPDATE games SET winner_id = $1, status = $2, total_moves = $3, duration_seconds = $4, completed_at = $5 WHERE id = $6`
	_, err := d.db.Exec(query, winnerID, status, totalMoves, duration, completedAt, gameID)
	if err != nil {
		return fmt.Errorf("failed to complete game: %w", err)
	}
	logger.Log.Info("Game completed", zap.String("game_id", gameID.String()), zap.String("status", string(status)))
	return nil
}

func (d *Database) SaveGameMove(gameID uuid.UUID, playerID, column, row, moveNumber int) error {
	query := `INSERT INTO game_moves (game_id, player_id, column_index, row_index, move_number) VALUES ($1, $2, $3, $4, $5)`
	_, err := d.db.Exec(query, gameID, playerID, column, row, moveNumber)
	if err != nil {
		return fmt.Errorf("failed to save game move: %w", err)
	}
	return nil
}

func (d *Database) GetLeaderboard(limit int) ([]models.LeaderboardEntry, error) {
	query := `SELECT id, username, games_won, games_played, win_rate, created_at FROM leaderboard LIMIT $1`
	rows, err := d.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get leaderboard: %w", err)
	}
	defer rows.Close()

	var entries []models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(&entry.ID, &entry.Username, &entry.GamesWon, &entry.GamesPlayed, &entry.WinRate, &entry.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}