package services
import (
	"connect4/internal/config"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ReconnectionService struct {
	config                *config.Config
	disconnectedPlayers   map[string]*models.DisconnectedPlayer
	disconnectedMutex     sync.RWMutex
	gameService           *GameService
	onForfeitCallback     func(gameID uuid.UUID, playerID int)
	onReconnectCallback   func(player *models.DisconnectedPlayer, gameState *models.GameState)
}

func NewReconnectionService(cfg *config.Config, gameService *GameService) *ReconnectionService {
	return &ReconnectionService{
		config:              cfg,
		disconnectedPlayers: make(map[string]*models.DisconnectedPlayer),
		gameService:         gameService,
	}
}

func (rs *ReconnectionService) SetForfeitCallback(callback func(gameID uuid.UUID, playerID int)) {
	rs.onForfeitCallback = callback
}

func (rs *ReconnectionService) SetReconnectCallback(callback func(player *models.DisconnectedPlayer, gameState *models.GameState)) {
	rs.onReconnectCallback = callback
}

func (rs *ReconnectionService) TrackDisconnection(username string, playerID int, gameID uuid.UUID) {
	rs.disconnectedMutex.Lock()
	defer rs.disconnectedMutex.Unlock()

	disconnectedPlayer := &models.DisconnectedPlayer{
		PlayerID:       playerID,
		Username:       username,
		GameID:         gameID,
		DisconnectedAt: time.Now(),
	}
	rs.disconnectedPlayers[username] = disconnectedPlayer

	go rs.startForfeitTimer(username)
	logger.Log.Info("Player disconnected", zap.String("username", username), zap.String("game_id", gameID.String()))
}

func (rs *ReconnectionService) startForfeitTimer(username string) {
	timeout := time.Duration(rs.config.Game.ReconnectionTimeout) * time.Second
	time.Sleep(timeout)

	rs.disconnectedMutex.Lock()
	defer rs.disconnectedMutex.Unlock()

	player, exists := rs.disconnectedPlayers[username]
	if exists {
		delete(rs.disconnectedPlayers, username)
		_ = rs.gameService.ForfeitGame(player.GameID, player.PlayerID)
		if rs.onForfeitCallback != nil {
			rs.onForfeitCallback(player.GameID, player.PlayerID)
		}
		logger.Log.Info("Player forfeited due to timeout", zap.String("username", username))
	}
}

func (rs *ReconnectionService) HandleReconnection(username string) (*models.GameState, error) {
	rs.disconnectedMutex.Lock()
	defer rs.disconnectedMutex.Unlock()

	player, exists := rs.disconnectedPlayers[username]
	if !exists {
		return nil, nil
	}

	gameState, err := rs.gameService.GetGame(player.GameID)
	if err != nil {
		return nil, err
	}

	delete(rs.disconnectedPlayers, username)
	if rs.onReconnectCallback != nil {
		rs.onReconnectCallback(player, gameState)
	}

	logger.Log.Info("Player reconnected", zap.String("username", username), zap.String("game_id", player.GameID.String()))
	return gameState, nil
}