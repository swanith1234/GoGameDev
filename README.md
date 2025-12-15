# Connect 4 - Real-time Multiplayer Game Backend

## 🚀 Features
- Real-time WebSocket-based gameplay
- Automatic matchmaking with 10-second bot fallback
- Competitive bot AI using Minimax algorithm
- Player reconnection (30-second window)
- Persistent game state in PostgreSQL (Supabase)
- Leaderboard system

## 📋 Prerequisites
- Go 1.21+
- PostgreSQL (Supabase)

## 🛠️ Setup

1. Install dependencies:
```bash
go mod tidy
```

2. Configure environment:
```bash
cp .env.example .env
# Edit .env with your Supabase credentials
```

3. Run database migrations:
```bash
psql $DATABASE_URL -f migrations/schema.sql
```

4. Run the server:
```bash
go run cmd/server/main.go
```

## 🔌 API Endpoints

### WebSocket
- `ws://localhost:8080/ws` - Game WebSocket connection

### REST
- `GET /api/health` - Health check
- `GET /api/leaderboard` - Get top 100 players

## 📦 WebSocket Events

### Client → Server
- `join-matchmaking` - Join matchmaking queue
- `make-move` - Make a game move

### Server → Client
- `game-started` - Game has started
- `move-accepted` - Your move was accepted
- `opponent-moved` - Opponent made a move
- `game-over` - Game ended
- `error` - Error occurred

## 🏗️ Project Structure
```
connect4/
├── cmd/server/          # Application entry point
├── internal/
│   ├── bot/            # Bot AI (Minimax)
│   ├── config/         # Configuration
│   ├── database/       # Database operations
│   ├── handlers/       # HTTP/WebSocket handlers
│   ├── models/         # Data models
│   └── services/       # Business logic
├── pkg/logger/         # Logging utilities
└── migrations/         # Database migrations
```

## 🚢 Deployment
Ready to deploy to Render, Railway, or Fly.io.

## 📝 License
MIT
