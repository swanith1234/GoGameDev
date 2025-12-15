package bot

import (
	"connect4/internal/models"
	"math"
)

const (
	maxDepth = 5
	botNum   = 2
	humanNum = 1
)

type Bot struct{}

func New() *Bot {
	return &Bot{}
}

func (b *Bot) GetBestMove(board models.Board) int {
	if col := b.findWinningMove(board, botNum); col != -1 {
		return col
	}
	if col := b.findWinningMove(board, humanNum); col != -1 {
		return col
	}

	bestScore := math.Inf(-1)
	bestCol := -1

	for col := 0; col < 7; col++ {
		if !board.IsValidMove(col) {
			continue
		}
		boardCopy := board.Copy()
		boardCopy.DropDisc(col, botNum)
		score := b.minimax(boardCopy, maxDepth-1, math.Inf(-1), math.Inf(1), false)
		if col == 3 {
			score += 0.1
		}
		if score > bestScore {
			bestScore = score
			bestCol = col
		}
	}

	if bestCol == -1 {
		if board.IsValidMove(3) {
			return 3
		}
		for col := 0; col < 7; col++ {
			if board.IsValidMove(col) {
				return col
			}
		}
	}
	return bestCol
}

func (b *Bot) findWinningMove(board models.Board, playerNum int) int {
	for col := 0; col < 7; col++ {
		if !board.IsValidMove(col) {
			continue
		}
		boardCopy := board.Copy()
		row := boardCopy.DropDisc(col, playerNum)
		if boardCopy.CheckWin(row, col) {
			return col
		}
	}
	return -1
}

func (b *Bot) minimax(board models.Board, depth int, alpha, beta float64, isMaximizing bool) float64 {
	if depth == 0 || board.IsFull() {
		return b.evaluateBoard(board)
	}

	if isMaximizing {
		maxEval := math.Inf(-1)
		for col := 0; col < 7; col++ {
			if !board.IsValidMove(col) {
				continue
			}
			boardCopy := board.Copy()
			row := boardCopy.DropDisc(col, botNum)
			if boardCopy.CheckWin(row, col) {
				return 1000.0 + float64(depth)
			}
			eval := b.minimax(boardCopy, depth-1, alpha, beta, false)
			maxEval = math.Max(maxEval, eval)
			alpha = math.Max(alpha, eval)
			if beta <= alpha {
				break
			}
		}
		return maxEval
	} else {
		minEval := math.Inf(1)
		for col := 0; col < 7; col++ {
			if !board.IsValidMove(col) {
				continue
			}
			boardCopy := board.Copy()
			row := boardCopy.DropDisc(col, humanNum)
			if boardCopy.CheckWin(row, col) {
				return -1000.0 - float64(depth)
			}
			eval := b.minimax(boardCopy, depth-1, alpha, beta, true)
			minEval = math.Min(minEval, eval)
			beta = math.Min(beta, eval)
			if beta <= alpha {
				break
			}
		}
		return minEval
	}
}

func (b *Bot) evaluateBoard(board models.Board) float64 {
	score := 0.0
	for row := 0; row < 6; row++ {
		for col := 0; col < 4; col++ {
			window := []int{board[row][col], board[row][col+1], board[row][col+2], board[row][col+3]}
			score += b.evaluateWindow(window)
		}
	}
	for col := 0; col < 7; col++ {
		for row := 0; row < 3; row++ {
			window := []int{board[row][col], board[row+1][col], board[row+2][col], board[row+3][col]}
			score += b.evaluateWindow(window)
		}
	}
	for row := 3; row < 6; row++ {
		for col := 0; col < 4; col++ {
			window := []int{board[row][col], board[row-1][col+1], board[row-2][col+2], board[row-3][col+3]}
			score += b.evaluateWindow(window)
		}
	}
	for row := 0; row < 3; row++ {
		for col := 0; col < 4; col++ {
			window := []int{board[row][col], board[row+1][col+1], board[row+2][col+2], board[row+3][col+3]}
			score += b.evaluateWindow(window)
		}
	}
	return score
}

func (b *Bot) evaluateWindow(window []int) float64 {
	score := 0.0
	botCount, humanCount, emptyCount := 0, 0, 0
	for _, cell := range window {
		if cell == botNum {
			botCount++
		} else if cell == humanNum {
			humanCount++
		} else {
			emptyCount++
		}
	}
	if botCount == 4 {
		score += 100
	} else if botCount == 3 && emptyCount == 1 {
		score += 10
	} else if botCount == 2 && emptyCount == 2 {
		score += 5
	}
	if humanCount == 3 && emptyCount == 1 {
		score -= 80
	}
	return score
}