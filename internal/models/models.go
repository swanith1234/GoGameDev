package models

import (
	"time"

	"github.com/google/uuid"
)

type Player struct {
	ID          int       `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	GamesPlayed int       `json:"games_played" db:"games_played"`
	GamesWon    int       `json:"games_won" db:"games_won"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type GameStatus string

const (
	GameStatusActive    GameStatus = "active"
	GameStatusCompleted GameStatus = "completed"
	GameStatusForfeited GameStatus = "forfeited"
	GameStatusDraw      GameStatus = "draw"
)

type Game struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	Player1ID       int        `json:"player1_id" db:"player1_id"`
	Player2ID       *int       `json:"player2_id" db:"player2_id"`
	Player2IsBot    bool       `json:"player2_is_bot" db:"player2_is_bot"`
	WinnerID        *int       `json:"winner_id" db:"winner_id"`
	Status          GameStatus `json:"status" db:"status"`
	DurationSeconds *int       `json:"duration_seconds" db:"duration_seconds"`
	TotalMoves      int        `json:"total_moves" db:"total_moves"`
	StartedAt       time.Time  `json:"started_at" db:"started_at"`
	CompletedAt     *time.Time `json:"completed_at" db:"completed_at"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}

type PlayerColor string

const (
	ColorRed    PlayerColor = "red"
	ColorYellow PlayerColor = "yellow"
)

type PlayerInfo struct {
	ID       int         `json:"id"`
	Username string      `json:"username"`
	Color    PlayerColor `json:"color"`
	IsBot    bool        `json:"is_bot"`
	SocketID string      `json:"socket_id,omitempty"`
}

type Board [6][7]int

type GameState struct {
	GameID      uuid.UUID   `json:"game_id"`
	Player1     PlayerInfo  `json:"player1"`
	Player2     PlayerInfo  `json:"player2"`
	Board       Board       `json:"board"`
	CurrentTurn PlayerColor `json:"current_turn"`
	Status      GameStatus  `json:"status"`
	Winner      *string     `json:"winner,omitempty"`
	MoveCount   int         `json:"move_count"`
	StartedAt   time.Time   `json:"started_at"`
	CompletedAt *time.Time  `json:"completed_at,omitempty"`
}

type WaitingPlayer struct {
	Username  string    `json:"username"`
	PlayerID  int       `json:"player_id"`
	SocketID  string    `json:"socket_id"`
	JoinedAt  time.Time `json:"joined_at"`
	TimerDone bool      `json:"timer_done"`
}

type DisconnectedPlayer struct {
	PlayerID       int       `json:"player_id"`
	Username       string    `json:"username"`
	GameID         uuid.UUID `json:"game_id"`
	DisconnectedAt time.Time `json:"disconnected_at"`
}

type LeaderboardEntry struct {
	ID          int       `json:"id" db:"id"`
	Username    string    `json:"username" db:"username"`
	GamesWon    int       `json:"games_won" db:"games_won"`
	GamesPlayed int       `json:"games_played" db:"games_played"`
	WinRate     float64   `json:"win_rate" db:"win_rate"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type WSMessageType string

const (
	WSJoinMatchmaking      WSMessageType = "join-matchmaking"
	WSMakeMove             WSMessageType = "make-move"
	WSReconnectGame        WSMessageType = "reconnect-game"
	WSGameStarted          WSMessageType = "game-started"
	WSMoveAccepted         WSMessageType = "move-accepted"
	WSOpponentMoved        WSMessageType = "opponent-moved"
	WSGameOver             WSMessageType = "game-over"
	WSOpponentDisconnected WSMessageType = "opponent-disconnected"
	WSOpponentReconnected  WSMessageType = "opponent-reconnected"
	WSGameRestored         WSMessageType = "game-restored"
	WSError                WSMessageType = "error"
	WSMatchmakingStatus    WSMessageType = "matchmaking-status"
)

type WSMessage struct {
	Type    WSMessageType `json:"type"`
	Payload interface{}   `json:"payload"`
}

type JoinMatchmakingPayload struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
}

type MakeMovePayload struct {
	GameID uuid.UUID `json:"game_id" binding:"required"`
	Column int       `json:"column" binding:"required,min=0,max=6"`
}

type GameStartedPayload struct {
	GameID      uuid.UUID   `json:"game_id"`
	Opponent    string      `json:"opponent"`
	YourColor   PlayerColor `json:"your_color"`
	CurrentTurn PlayerColor `json:"current_turn"`
	IsBot       bool        `json:"is_bot"`
}

type MovePayload struct {
	Column     int         `json:"column"`
	Row        int         `json:"row"`
	Color      PlayerColor `json:"color"`
	NextTurn   PlayerColor `json:"next_turn"`
	Board      Board       `json:"board"`
	MoveNumber int         `json:"move_number"`
}

type GameOverPayload struct {
	Winner   *string `json:"winner"`
	Reason   string  `json:"reason"`
	Board    Board   `json:"board"`
	Duration int     `json:"duration_seconds"`
}

type ErrorPayload struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

func NewBoard() Board {
	return Board{}
}

func (b *Board) IsValidMove(column int) bool {
	if column < 0 || column > 6 {
		return false
	}
	return b[0][column] == 0
}

func (b *Board) DropDisc(column int, playerNum int) int {
	for row := 5; row >= 0; row-- {
		if b[row][column] == 0 {
			b[row][column] = playerNum
			return row
		}
	}
	return -1
}

func (b *Board) CheckWin(row, col int) bool {
	player := b[row][col]
	if player == 0 {
		return false
	}

	directions := [][2]int{{0, 1}, {1, 0}, {1, 1}, {-1, 1}}
	for _, dir := range directions {
		if b.checkDirection(row, col, dir[0], dir[1], player) {
			return true
		}
	}
	return false
}

func (b *Board) checkDirection(row, col, dRow, dCol, player int) bool {
	count := 1
	r, c := row+dRow, col+dCol
	for r >= 0 && r < 6 && c >= 0 && c < 7 && b[r][c] == player {
		count++
		r += dRow
		c += dCol
	}
	r, c = row-dRow, col-dCol
	for r >= 0 && r < 6 && c >= 0 && c < 7 && b[r][c] == player {
		count++
		r -= dRow
		c -= dCol
	}
	return count >= 4
}

func (b *Board) IsFull() bool {
	for col := 0; col < 7; col++ {
		if b[0][col] == 0 {
			return false
		}
	}
	return true
}

func (b *Board) Copy() Board {
	var newBoard Board
	for i := 0; i < 6; i++ {
		for j := 0; j < 7; j++ {
			newBoard[i][j] = b[i][j]
		}
	}
	return newBoard
}