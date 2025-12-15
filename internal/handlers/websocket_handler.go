package handlers

import (
	"connect4/internal/models"
	"connect4/internal/services"
	"connect4/pkg/logger"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type WSHandler struct {
	matchmakingService  *services.MatchmakingService
	gameService         *services.GameService
	reconnectionService *services.ReconnectionService
	connections         map[string]*websocket.Conn
	playerGames         map[string]uuid.UUID
	connMutex           sync.RWMutex
}

func NewWSHandler(matchmaking *services.MatchmakingService, game *services.GameService, reconnection *services.ReconnectionService) *WSHandler {
	handler := &WSHandler{
		matchmakingService:  matchmaking,
		gameService:         game,
		reconnectionService: reconnection,
		connections:         make(map[string]*websocket.Conn),
		playerGames:         make(map[string]uuid.UUID),
	}

	matchmaking.SetMatchCallback(handler.handlePlayerMatch)
	matchmaking.SetBotCallback(handler.handleBotMatch)
	reconnection.SetForfeitCallback(handler.handleForfeit)
	reconnection.SetReconnectCallback(handler.handleReconnect)

	return handler
}

func (h *WSHandler) HandleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		logger.Log.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	socketID := uuid.New().String()
	defer conn.Close()

	var username string

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if username != "" {
				h.handleDisconnection(username)
			}
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			h.sendError(conn, "Invalid message format")
			continue
		}

		switch wsMsg.Type {
		case models.WSJoinMatchmaking:
			username = h.handleJoinMatchmaking(conn, socketID, wsMsg.Payload)
		case models.WSMakeMove:
			h.handleMakeMove(conn, username, wsMsg.Payload)
		case models.WSReconnectGame:
			h.handleReconnectGame(conn, username, wsMsg.Payload)
		}
	}
}

func (h *WSHandler) handleJoinMatchmaking(conn *websocket.Conn, socketID string, payload interface{}) string {
	data, _ := json.Marshal(payload)
	var joinPayload models.JoinMatchmakingPayload
	if err := json.Unmarshal(data, &joinPayload); err != nil || joinPayload.Username == "" {
		h.sendError(conn, "Invalid username")
		return ""
	}

	username := joinPayload.Username
	h.connMutex.Lock()
	h.connections[username] = conn
	h.connMutex.Unlock()

	if err := h.matchmakingService.JoinQueue(username, socketID); err != nil {
		h.sendError(conn, err.Error())
		return username
	}

	h.sendMessage(conn, models.WSMessage{
		Type:    models.WSMatchmakingStatus,
		Payload: map[string]interface{}{"status": "searching", "message": "Looking for opponent..."},
	})

	return username
}

func (h *WSHandler) handlePlayerMatch(player1, player2 *models.WaitingPlayer, gameState *models.GameState) {
	h.connMutex.Lock()
	conn1 := h.connections[player1.Username]
	conn2 := h.connections[player2.Username]
	h.playerGames[player1.Username] = gameState.GameID
	h.playerGames[player2.Username] = gameState.GameID
	h.connMutex.Unlock()

	if conn1 != nil {
		h.sendMessage(conn1, models.WSMessage{
			Type: models.WSGameStarted,
			Payload: models.GameStartedPayload{
				GameID:      gameState.GameID,
				Opponent:    player2.Username,
				YourColor:   models.ColorRed,
				CurrentTurn: models.ColorRed,
				IsBot:       false,
			},
		})
	}

	if conn2 != nil {
		h.sendMessage(conn2, models.WSMessage{
			Type: models.WSGameStarted,
			Payload: models.GameStartedPayload{
				GameID:      gameState.GameID,
				Opponent:    player1.Username,
				YourColor:   models.ColorYellow,
				CurrentTurn: models.ColorRed,
				IsBot:       false,
			},
		})
	}
}

func (h *WSHandler) handleBotMatch(player *models.WaitingPlayer, gameState *models.GameState) {
	h.connMutex.Lock()
	conn := h.connections[player.Username]
	h.playerGames[player.Username] = gameState.GameID
	h.connMutex.Unlock()

	if conn != nil {
		h.sendMessage(conn, models.WSMessage{
			Type: models.WSGameStarted,
			Payload: models.GameStartedPayload{
				GameID:      gameState.GameID,
				Opponent:    "Bot",
				YourColor:   models.ColorRed,
				CurrentTurn: models.ColorRed,
				IsBot:       true,
			},
		})
	}
}

func (h *WSHandler) handleMakeMove(conn *websocket.Conn, username string, payload interface{}) {
	data, _ := json.Marshal(payload)
	var movePayload models.MakeMovePayload
	if err := json.Unmarshal(data, &movePayload); err != nil {
		h.sendError(conn, "Invalid move payload")
		return
	}

	game, err := h.gameService.GetGame(movePayload.GameID)
	if err != nil {
		h.sendError(conn, "Game not found")
		return
	}

	var playerID int
	if game.Player1.Username == username {
		playerID = game.Player1.ID
	} else if game.Player2.Username == username {
		playerID = game.Player2.ID
	} else {
		h.sendError(conn, "You are not in this game")
		return
	}

	move, gameOver, err := h.gameService.MakeMove(movePayload.GameID, playerID, movePayload.Column)
	if err != nil {
		h.sendError(conn, err.Error())
		return
	}

	h.sendMessage(conn, models.WSMessage{Type: models.WSMoveAccepted, Payload: move})

	opponentUsername := game.Player2.Username
	if username == game.Player2.Username {
		opponentUsername = game.Player1.Username
	}
	h.connMutex.RLock()
	opponentConn := h.connections[opponentUsername]
	h.connMutex.RUnlock()
	if opponentConn != nil {
		h.sendMessage(opponentConn, models.WSMessage{Type: models.WSOpponentMoved, Payload: move})
	}

	if gameOver != nil {
		h.sendMessage(conn, models.WSMessage{Type: models.WSGameOver, Payload: gameOver})
		if opponentConn != nil && !game.Player2.IsBot {
			h.sendMessage(opponentConn, models.WSMessage{Type: models.WSGameOver, Payload: gameOver})
		}
		return
	}

	if game.Player2.IsBot && move.NextTurn == models.ColorYellow {
		time.Sleep(500 * time.Millisecond)
		botMove, botGameOver, err := h.gameService.MakeBotMove(movePayload.GameID)
		if err == nil {
			h.sendMessage(conn, models.WSMessage{Type: models.WSOpponentMoved, Payload: botMove})
			if botGameOver != nil {
				h.sendMessage(conn, models.WSMessage{Type: models.WSGameOver, Payload: botGameOver})
			}
		}
	}
}

func (h *WSHandler) handleReconnectGame(conn *websocket.Conn, username string, payload interface{}) {
	gameState, err := h.reconnectionService.HandleReconnection(username)
	if err != nil || gameState == nil {
		h.sendError(conn, "Failed to reconnect to game")
		return
	}

	h.connMutex.Lock()
	h.connections[username] = conn
	h.playerGames[username] = gameState.GameID
	h.connMutex.Unlock()

	var yourColor models.PlayerColor
	var opponentName string
	if gameState.Player1.Username == username {
		yourColor = gameState.Player1.Color
		opponentName = gameState.Player2.Username
	} else {
		yourColor = gameState.Player2.Color
		opponentName = gameState.Player1.Username
	}

	h.sendMessage(conn, models.WSMessage{
		Type: models.WSGameRestored,
		Payload: map[string]interface{}{
			"game_id":      gameState.GameID,
			"board":        gameState.Board,
			"current_turn": gameState.CurrentTurn,
			"move_count":   gameState.MoveCount,
			"your_color":   yourColor,
			"opponent":     opponentName,
		},
	})

	// Notify opponent
	h.connMutex.RLock()
	opponentConn := h.connections[opponentName]
	h.connMutex.RUnlock()

	if opponentConn != nil {
		h.sendMessage(opponentConn, models.WSMessage{
			Type: models.WSOpponentReconnected,
			Payload: map[string]interface{}{
				"message": username + " has reconnected",
			},
		})
	}
}

func (h *WSHandler) handleDisconnection(username string) {
	h.connMutex.Lock()
	delete(h.connections, username)
	gameID, hasGame := h.playerGames[username]
	h.connMutex.Unlock()

	if hasGame {
		game, err := h.gameService.GetGame(gameID)
		if err == nil && game.Status == models.GameStatusActive {
			var playerID int
			if game.Player1.Username == username {
				playerID = game.Player1.ID
			} else {
				playerID = game.Player2.ID
			}
			h.reconnectionService.TrackDisconnection(username, playerID, gameID)

			// Notify opponent
			opponentUsername := game.Player2.Username
			if username == game.Player2.Username {
				opponentUsername = game.Player1.Username
			}
			h.connMutex.RLock()
			opponentConn := h.connections[opponentUsername]
			h.connMutex.RUnlock()
			if opponentConn != nil {
				h.sendMessage(opponentConn, models.WSMessage{
					Type: models.WSOpponentDisconnected,
					Payload: map[string]interface{}{
						"time_remaining": 30,
					},
				})
			}
		}
	} else {
		h.matchmakingService.LeaveQueue(username)
	}
}

func (h *WSHandler) handleForfeit(gameID uuid.UUID, playerID int) {
	game, err := h.gameService.GetGame(gameID)
	if err != nil {
		return
	}

	var winnerUsername string
	var loserUsername string
	if game.Player1.ID == playerID {
		winnerUsername = game.Player2.Username
		loserUsername = game.Player1.Username
	} else {
		winnerUsername = game.Player1.Username
		loserUsername = game.Player2.Username
	}

	h.connMutex.RLock()
	winnerConn := h.connections[winnerUsername]
	h.connMutex.RUnlock()

	if winnerConn != nil {
		h.sendMessage(winnerConn, models.WSMessage{
			Type: models.WSGameOver,
			Payload: models.GameOverPayload{
				Winner:   &winnerUsername,
				Reason:   "forfeit",
				Board:    game.Board,
				Duration: 30,
			},
		})
	}

	logger.Log.Info("Game forfeited due to disconnect", zap.String("loser", loserUsername), zap.String("winner", winnerUsername))
}

func (h *WSHandler) handleReconnect(player *models.DisconnectedPlayer, gameState *models.GameState) {
	// This is called by reconnection service when player reconnects
	// The actual reconnection handling is done in handleReconnectGame
	logger.Log.Info("Player reconnected successfully", zap.String("username", player.Username))
}

func (h *WSHandler) sendMessage(conn *websocket.Conn, msg models.WSMessage) {
	if err := conn.WriteJSON(msg); err != nil {
		logger.Log.Error("Failed to send message", zap.Error(err))
	}
}

func (h *WSHandler) sendError(conn *websocket.Conn, message string) {
	h.sendMessage(conn, models.WSMessage{
		Type:    models.WSError,
		Payload: models.ErrorPayload{Message: message},
	})
}
