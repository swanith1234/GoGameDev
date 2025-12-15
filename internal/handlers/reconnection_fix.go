package handlers

// Add this to websocket_handler.go - Update handleReconnect method

func (h *WSHandler) handleReconnect(player *models.DisconnectedPlayer, gameState *models.GameState) {
	h.connMutex.RLock()
	conn := h.connections[player.Username]
	h.connMutex.RUnlock()

	if conn != nil {
		// Send game restored message with full state
		h.sendMessage(conn, models.WSMessage{
			Type: models.WSGameRestored,
			Payload: map[string]interface{}{
				"game_id":      gameState.GameID,
				"board":        gameState.Board,
				"current_turn": gameState.CurrentTurn,
				"move_count":   gameState.MoveCount,
				"your_color":   getPlayerColor(player.Username, gameState),
				"opponent":     getOpponentName(player.Username, gameState),
			},
		})

		// Notify opponent of reconnection
		opponentName := getOpponentName(player.Username, gameState)
		h.connMutex.RLock()
		opponentConn := h.connections[opponentName]
		h.connMutex.RUnlock()

		if opponentConn != nil {
			h.sendMessage(opponentConn, models.WSMessage{
				Type: models.WSOpponentReconnected,
				Payload: map[string]interface{}{
					"message": player.Username + " has reconnected",
				},
			})
		}
	}
}

func getPlayerColor(username string, game *models.GameState) models.PlayerColor {
	if game.Player1.Username == username {
		return game.Player1.Color
	}
	return game.Player2.Color
}

func getOpponentName(username string, game *models.GameState) string {
	if game.Player1.Username == username {
		return game.Player2.Username
	}
	return game.Player1.Username
}
