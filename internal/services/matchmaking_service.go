package services
import (
	"connect4/internal/config"
	"connect4/internal/database"
	"connect4/internal/models"
	"connect4/pkg/logger"
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

type MatchmakingService struct {
	db              *database.Database
	config          *config.Config
	waitingQueue    []*models.WaitingPlayer
	queueMutex      sync.Mutex
	gameService     *GameService
	onMatchCallback func(player1, player2 *models.WaitingPlayer, gameState *models.GameState)
	onBotCallback   func(player *models.WaitingPlayer, gameState *models.GameState)
}

func NewMatchmakingService(db *database.Database, cfg *config.Config, gameService *GameService) *MatchmakingService {
	return &MatchmakingService{
		db:           db,
		config:       cfg,
		waitingQueue: make([]*models.WaitingPlayer, 0),
		gameService:  gameService,
	}
}

func (ms *MatchmakingService) SetMatchCallback(callback func(player1, player2 *models.WaitingPlayer, gameState *models.GameState)) {
	ms.onMatchCallback = callback
}

func (ms *MatchmakingService) SetBotCallback(callback func(player *models.WaitingPlayer, gameState *models.GameState)) {
	ms.onBotCallback = callback
}

func (ms *MatchmakingService) JoinQueue(username, socketID string) error {
	ms.queueMutex.Lock()
	defer ms.queueMutex.Unlock()

	for _, p := range ms.waitingQueue {
		if p.Username == username {
			return errors.New("player already in queue")
		}
	}

	player, err := ms.db.GetPlayerByUsername(username)
	if err != nil {
		return err
	}
	if player == nil {
		player, err = ms.db.CreatePlayer(username)
		if err != nil {
			return err
		}
	}

	waitingPlayer := &models.WaitingPlayer{
		Username:  username,
		PlayerID:  player.ID,
		SocketID:  socketID,
		JoinedAt:  time.Now(),
		TimerDone: false,
	}

	if len(ms.waitingQueue) > 0 {
		opponent := ms.waitingQueue[0]
		ms.waitingQueue = ms.waitingQueue[1:]
		go ms.createMatch(opponent, waitingPlayer)
		logger.Log.Info("Players matched", zap.String("player1", opponent.Username), zap.String("player2", waitingPlayer.Username))
		return nil
	}

	ms.waitingQueue = append(ms.waitingQueue, waitingPlayer)
	go ms.startBotTimer(waitingPlayer)
	logger.Log.Info("Player joined matchmaking queue", zap.String("username", username))
	return nil
}

func (ms *MatchmakingService) startBotTimer(player *models.WaitingPlayer) {
	timeout := time.Duration(ms.config.Game.MatchmakingTimeout) * time.Second
	time.Sleep(timeout)

	ms.queueMutex.Lock()
	defer ms.queueMutex.Unlock()

	for i, p := range ms.waitingQueue {
		if p.Username == player.Username && !p.TimerDone {
			ms.waitingQueue = append(ms.waitingQueue[:i], ms.waitingQueue[i+1:]...)
			go ms.createBotMatch(player)
			logger.Log.Info("Matchmaking timeout - starting bot game", zap.String("player", player.Username))
			return
		}
	}
}

func (ms *MatchmakingService) createMatch(player1, player2 *models.WaitingPlayer) {
	player1Info := models.PlayerInfo{
		ID:       player1.PlayerID,
		Username: player1.Username,
		Color:    models.ColorRed,
		IsBot:    false,
		SocketID: player1.SocketID,
	}
	player2Info := models.PlayerInfo{
		ID:       player2.PlayerID,
		Username: player2.Username,
		Color:    models.ColorYellow,
		IsBot:    false,
		SocketID: player2.SocketID,
	}
	gameState, err := ms.gameService.CreateGame(player1Info, player2Info)
	if err != nil {
		logger.Log.Error("Failed to create game", zap.Error(err))
		return
	}
	if ms.onMatchCallback != nil {
		ms.onMatchCallback(player1, player2, gameState)
	}
}

func (ms *MatchmakingService) createBotMatch(player *models.WaitingPlayer) {
	playerInfo := models.PlayerInfo{
		ID:       player.PlayerID,
		Username: player.Username,
		Color:    models.ColorRed,
		IsBot:    false,
		SocketID: player.SocketID,
	}
	botPlayer, err := ms.db.CreatePlayer("Bot_" + time.Now().Format("20060102150405"))
	if err != nil {
		logger.Log.Error("Failed to create bot player", zap.Error(err))
		return
	}
	botInfo := models.PlayerInfo{
		ID:       botPlayer.ID,
		Username: "Bot",
		Color:    models.ColorYellow,
		IsBot:    true,
	}
	gameState, err := ms.gameService.CreateGame(playerInfo, botInfo)
	if err != nil {
		logger.Log.Error("Failed to create bot game", zap.Error(err))
		return
	}
	if ms.onBotCallback != nil {
		ms.onBotCallback(player, gameState)
	}
}

func (ms *MatchmakingService) LeaveQueue(username string) {
	ms.queueMutex.Lock()
	defer ms.queueMutex.Unlock()
	for i, p := range ms.waitingQueue {
		if p.Username == username {
			ms.waitingQueue = append(ms.waitingQueue[:i], ms.waitingQueue[i+1:]...)
			return
		}
	}
}

