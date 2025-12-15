package services

import (
	"connect4/internal/bot"
	"connect4/internal/database"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type GameService struct {
	db          *database.Database
	activeGames map[uuid.UUID]*models.GameState
	gamesMutex  sync.RWMutex
	bot         *bot.Bot
}

func NewGameService(db *database.Database) *GameService {
	return &GameService{
		db:          db,
		activeGames: make(map[uuid.UUID]*models.GameState),
		bot:         bot.New(),
	}
}

func (gs *GameService) CreateGame(player1 models.PlayerInfo, player2 models.PlayerInfo) (*models.GameState, error) {
	var player2ID *int
	if !player2.IsBot {
		player2ID = &player2.ID
	}

	dbGameID, err := gs.db.CreateGame(player1.ID, player2ID, player2.IsBot)
	if err != nil {
		return nil, fmt.Errorf("failed to create game in database: %w", err)
	}

	gameState := &models.GameState{
		GameID:      dbGameID,
		Player1:     player1,
		Player2:     player2,
		Board:       models.NewBoard(),
		CurrentTurn: models.ColorRed,
		Status:      models.GameStatusActive,
		MoveCount:   0,
		StartedAt:   time.Now(),
	}

	gs.gamesMutex.Lock()
	gs.activeGames[dbGameID] = gameState
	gs.gamesMutex.Unlock()

	logger.Log.Info("Game created", zap.String("game_id", dbGameID.String()), zap.String("player1", player1.Username), zap.String("player2", player2.Username), zap.Bool("is_bot", player2.IsBot))
	return gameState, nil
}

func (gs *GameService) GetGame(gameID uuid.UUID) (*models.GameState, error) {
	gs.gamesMutex.RLock()
	defer gs.gamesMutex.RUnlock()
	game, exists := gs.activeGames[gameID]
	if !exists {
		return nil, errors.New("game not found")
	}
	return game, nil
}

func (gs *GameService) MakeMove(gameID uuid.UUID, playerID int, column int) (*models.MovePayload, *models.GameOverPayload, error) {
	gs.gamesMutex.Lock()
	defer gs.gamesMutex.Unlock()

	game, exists := gs.activeGames[gameID]
	if !exists {
		return nil, nil, errors.New("game not found")
	}
	if game.Status != models.GameStatusActive {
		return nil, nil, errors.New("game is not active")
	}

	currentPlayer := game.Player1
	if game.CurrentTurn == models.ColorYellow {
		currentPlayer = game.Player2
	}
	if currentPlayer.ID != playerID {
		return nil, nil, errors.New("not your turn")
	}
	if !game.Board.IsValidMove(column) {
		return nil, nil, errors.New("invalid move: column is full")
	}

	playerNum := 1
	if game.CurrentTurn == models.ColorYellow {
		playerNum = 2
	}
	row := game.Board.DropDisc(column, playerNum)
	if row == -1 {
		return nil, nil, errors.New("failed to drop disc")
	}
	game.MoveCount++

	_ = gs.db.SaveGameMove(gameID, playerID, column, row, game.MoveCount)

	if game.Board.CheckWin(row, column) {
		return gs.handleGameEnd(game, &currentPlayer.ID, "win", column, row, currentPlayer.Color)
	}
	if game.Board.IsFull() {
		return gs.handleGameEnd(game, nil, "draw", column, row, currentPlayer.Color)
	}

	if game.CurrentTurn == models.ColorRed {
		game.CurrentTurn = models.ColorYellow
	} else {
		game.CurrentTurn = models.ColorRed
	}

	movePayload := &models.MovePayload{
		Column:     column,
		Row:        row,
		Color:      currentPlayer.Color,
		NextTurn:   game.CurrentTurn,
		Board:      game.Board,
		MoveNumber: game.MoveCount,
	}
	return movePayload, nil, nil
}

func (gs *GameService) MakeBotMove(gameID uuid.UUID) (*models.MovePayload, *models.GameOverPayload, error) {
	gs.gamesMutex.Lock()
	defer gs.gamesMutex.Unlock()

	game, exists := gs.activeGames[gameID]
	if !exists {
		return nil, nil, errors.New("game not found")
	}
	if game.Status != models.GameStatusActive {
		return nil, nil, errors.New("game is not active")
	}
	if !game.Player2.IsBot {
		return nil, nil, errors.New("player 2 is not a bot")
	}
	if game.CurrentTurn != game.Player2.Color {
		return nil, nil, errors.New("not bot's turn")
	}

	column := gs.bot.GetBestMove(game.Board)
	row := game.Board.DropDisc(column, 2)
	if row == -1 {
		return nil, nil, errors.New("failed to drop disc")
	}
	game.MoveCount++

	_ = gs.db.SaveGameMove(gameID, game.Player2.ID, column, row, game.MoveCount)

	if game.Board.CheckWin(row, column) {
		return gs.handleGameEnd(game, &game.Player2.ID, "win", column, row, game.Player2.Color)
	}
	if game.Board.IsFull() {
		return gs.handleGameEnd(game, nil, "draw", column, row, game.Player2.Color)
	}

	game.CurrentTurn = models.ColorRed

	movePayload := &models.MovePayload{
		Column:     column,
		Row:        row,
		Color:      game.Player2.Color,
		NextTurn:   game.CurrentTurn,
		Board:      game.Board,
		MoveNumber: game.MoveCount,
	}
	return movePayload, nil, nil
}

func (gs *GameService) handleGameEnd(game *models.GameState, winnerID *int, reason string, column int, row int, color models.PlayerColor) (*models.MovePayload, *models.GameOverPayload, error) {
	completedAt := time.Now()
	game.CompletedAt = &completedAt

	var status models.GameStatus
	if reason == "draw" {
		status = models.GameStatusDraw
		game.Status = status
	} else if reason == "win" {
		status = models.GameStatusCompleted
		game.Status = status
		if winnerID != nil {
			if *winnerID == game.Player1.ID {
				game.Winner = &game.Player1.Username
			} else {
				game.Winner = &game.Player2.Username
			}
		}
	} else if reason == "forfeit" {
		status = models.GameStatusForfeited
		game.Status = status
	}

	_ = gs.db.CompleteGame(game.GameID, winnerID, status, game.MoveCount, game.StartedAt)

	movePayload := &models.MovePayload{
		Column:     column,
		Row:        row,
		Color:      color,
		Board:      game.Board,
		MoveNumber: game.MoveCount,
	}

	gameOverPayload := &models.GameOverPayload{
		Winner:   game.Winner,
		Reason:   reason,
		Board:    game.Board,
		Duration: int(completedAt.Sub(game.StartedAt).Seconds()),
	}

	return movePayload, gameOverPayload, nil
}

func (gs *GameService) ForfeitGame(gameID uuid.UUID, playerID int) error {
	gs.gamesMutex.Lock()
	defer gs.gamesMutex.Unlock()

	game, exists := gs.activeGames[gameID]
	if !exists {
		return errors.New("game not found")
	}

	var winnerID int
	if game.Player1.ID == playerID {
		winnerID = game.Player2.ID
	} else {
		winnerID = game.Player1.ID
	}

	completedAt := time.Now()
	game.CompletedAt = &completedAt
	game.Status = models.GameStatusForfeited

	if winnerID == game.Player1.ID {
		game.Winner = &game.Player1.Username
	} else {
		game.Winner = &game.Player2.Username
	}

	_ = gs.db.CompleteGame(game.GameID, &winnerID, models.GameStatusForfeited, game.MoveCount, game.StartedAt)
	return nil
}
