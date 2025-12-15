


DROP TABLE IF EXISTS game_moves CASCADE;
DROP TABLE IF EXISTS game_analytics CASCADE;
DROP TABLE IF EXISTS games CASCADE;
DROP TABLE IF EXISTS players CASCADE;
DROP VIEW IF EXISTS leaderboard;

-- Create players table
CREATE TABLE players (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    games_played INT DEFAULT 0,
    games_won INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create games table
CREATE TABLE games (
    id UUID PRIMARY KEY,
    player1_id INT NOT NULL REFERENCES players(id),
    player2_id INT REFERENCES players(id),
    player2_is_bot BOOLEAN DEFAULT FALSE,
    winner_id INT REFERENCES players(id),
    status VARCHAR(20) NOT NULL CHECK (status IN ('active', 'completed', 'forfeited', 'draw')),
    duration_seconds INT,
    total_moves INT DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create game_moves table
CREATE TABLE game_moves (
    id SERIAL PRIMARY KEY,
    game_id UUID NOT NULL REFERENCES games(id) ON DELETE CASCADE,
    player_id INT NOT NULL REFERENCES players(id),
    column_index INT NOT NULL CHECK (column_index >= 0 AND column_index <= 6),
    row_index INT NOT NULL CHECK (row_index >= 0 AND row_index <= 5),
    move_number INT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_games_player1 ON games(player1_id);
CREATE INDEX idx_games_player2 ON games(player2_id);
CREATE INDEX idx_games_status ON games(status);
CREATE INDEX idx_games_started_at ON games(started_at);
CREATE INDEX idx_game_moves_game_id ON game_moves(game_id);
CREATE INDEX idx_players_username ON players(username);

-- Create leaderboard view
CREATE OR REPLACE VIEW leaderboard AS
SELECT 
    p.id,
    p.username,
    p.games_won,
    p.games_played,
    CASE 
        WHEN p.games_played > 0 THEN ROUND((p.games_won::NUMERIC / p.games_played * 100), 2)
        ELSE 0 
    END as win_rate,
    p.created_at
FROM players p
WHERE p.games_played > 0
ORDER BY p.games_won DESC, win_rate DESC, p.games_played DESC
LIMIT 100;

-- Function to update player stats
CREATE OR REPLACE FUNCTION update_player_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE players 
    SET 
        games_played = games_played + 1,
        games_won = games_won + CASE WHEN NEW.winner_id = NEW.player1_id THEN 1 ELSE 0 END,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.player1_id;
    
    IF NEW.player2_id IS NOT NULL AND NEW.player2_is_bot = FALSE THEN
        UPDATE players 
        SET 
            games_played = games_played + 1,
            games_won = games_won + CASE WHEN NEW.winner_id = NEW.player2_id THEN 1 ELSE 0 END,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.player2_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update stats
CREATE TRIGGER trigger_update_player_stats
    AFTER UPDATE OF status ON games
    FOR EACH ROW
    WHEN (NEW.status IN ('completed', 'forfeited', 'draw') AND OLD.status = 'active')
    EXECUTE FUNCTION update_player_stats();

