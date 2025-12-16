# 🎮 Connect 4 - Production-Grade Real-Time Multiplayer Game

## 🌟 Features

### Core Features
- ✅ Real-time WebSocket gameplay
- ✅ Automatic player matchmaking (10-second timeout)
- ✅ Competitive bot AI (Minimax with Alpha-Beta Pruning)
- ✅ Player reconnection system (30-second window)
- ✅ Persistent game state (Supabase PostgreSQL)
- ✅ Real-time leaderboard

### Bonus Features (Production-Grade)
- ✅ Kafka event streaming for analytics
- ✅ Dedicated analytics consumer service
- ✅ Comprehensive game metrics and insights
- ✅ Player performance tracking
- ✅ Hourly/daily trend analysis
- ✅ Popular column analysis
- ✅ Win streak tracking

## 🏗️ Architecture
```
┌─────────────┐
│   Frontend  │
└──────┬──────┘
       │ WebSocket + HTTP
       ▼
┌──────────────────────────────────────┐
│        Go Backend Server              │
│  ┌────────────────────────────────┐  │
│  │  WebSocket Handler             │  │
│  │  Game Logic Service            │  │
│  │  Matchmaking Service           │  │
│  │  Bot AI (Minimax)              │  │
│  │  Kafka Producer                │  │
│  └────────────────────────────────┘  │
└──────────┬─────────────┬─────────────┘
           │             │
           ▼             ▼
    ┌──────────┐   ┌──────────────┐
    │ Supabase │   │    Kafka     │
    │PostgreSQL│   │   Cluster    │
    └──────────┘   └──────┬───────┘
                          │
                          ▼
                   ┌──────────────────┐
                   │ Analytics        │
                   │ Consumer Service │
                   └──────────────────┘
```

## 📋 Prerequisites

- Go 1.21+
- PostgreSQL (Supabase account)
- Kafka (optional - for analytics)
- Docker & Docker Compose (optional)

## 🚀 Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/yourusername/connect4-backend
cd connect4-backend
```

### 2. Configure Environment
```bash
cp .env.example .env
# Edit .env with your Supabase credentials
```

### 3. Install Dependencies
```bash
go mod download
go mod tidy
```

### 4. Run Database Migrations
```bash
psql "your-supabase-connection-string" -f migrations/schema.sql
```

### 5. Run Server
```bash
# Without Kafka (simple mode)
go run cmd/server/main.go

# With Kafka (production mode)
# Terminal 1: Start server
go run cmd/server/main.go

# Terminal 2: Start analytics consumer
go run cmd/analytics/main.go
```

## 🐳 Docker Deployment
```bash
# Build and run with Docker Compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

## 📡 API Endpoints

### WebSocket
- `ws://localhost:8080/ws` - Game WebSocket connection

### REST API

#### Game Endpoints
- `GET /health` - Health check
- `GET /api/leaderboard` - Get top 100 players
- `GET /api/player/:username` - Get player stats

#### Analytics Endpoints (Bonus)
- `GET /api/analytics/stats` - Overall game statistics
- `GET /api/analytics/popular-columns` - Most played columns
- `GET /api/analytics/hourly` - Hourly game distribution
- `GET /api/analytics/player/:username` - Detailed player performance
- `GET /api/analytics/trends` - Trending patterns

## 🎮 WebSocket Events

### Client → Server
```json
{
  "type": "join-matchmaking",
  "payload": { "username": "player1" }
}

{
  "type": "make-move",
  "payload": { "game_id": "uuid", "column": 3 }
}
```

### Server → Client
```json
{
  "type": "game-started",
  "payload": {
    "game_id": "uuid",
    "opponent": "player2",
    "your_color": "red",
    "current_turn": "red",
    "is_bot": false
  }
}
```

## 📊 Analytics Features

### Game Statistics
- Total games played
- Games today/this hour
- Active games
- Average game duration
- Bot vs Human win rates
- Draw rate
- Peak playing hours

### Player Analytics
- Win/loss records
- Average game time
- Favorite columns
- Win streaks
- Performance over time
- Bot vs Human performance

### Trending Insights
- Daily game trends
- Most active players
- Popular strategies
- Column usage patterns

## 🧪 Testing
```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Test specific package
go test ./internal/services -v

# Integration tests
go test ./tests/integration -v
```

## 🚢 Production Deployment

### Railway
```bash
railway init
railway up
```

### Render
1. Connect GitHub repository
2. Add environment variables
3. Deploy

### Fly.io
```bash
flyctl launch
flyctl deploy
```

## 📈 Performance Metrics

- **WebSocket Latency:** < 50ms
- **Bot Response Time:** < 500ms
- **Concurrent Games:** 1000+
- **Database Queries:** < 10ms average

## 🔐 Security Features

- Input validation
- SQL injection prevention
- CORS configuration
- Rate limiting ready
- Environment-based configuration
- Secure WebSocket connections

## 🛠️ Tech Stack

- **Language:** Go 1.21
- **Web Framework:** Gin
- **WebSocket:** Gorilla WebSocket
- **Database:** PostgreSQL (Supabase)
- **Message Queue:** Kafka
- **Logging:** Uber Zap
- **Containerization:** Docker

## 📝 Environment Variables
```env
# Server
PORT=8080
ENV=production

# Database
DB_HOST=db.xxxxx.supabase.co
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your_password
DB_NAME=postgres
DB_SSLMODE=require

# Kafka (Optional)
KAFKA_BROKERS=broker1:9092,broker2:9092
KAFKA_TOPIC_EVENTS=game.events

# Game
MATCHMAKING_TIMEOUT=10
RECONNECTION_TIMEOUT=30
```

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📄 License

MIT License - see LICENSE file for details

## 👨‍💻 Author

**Your Name**
- GitHub: [@yourusername](https://github.com/yourusername)
- LinkedIn: [Your LinkedIn](https://linkedin.com/in/yourprofile)
- Portfolio: [your-portfolio.com](https://your-portfolio.com)

## 🎯 Project Highlights

This project demonstrates:
- ✅ Production-grade Go architecture
- ✅ Real-time WebSocket communication
- ✅ Event-driven design with Kafka
- ✅ Strategic AI implementation (Minimax)
- ✅ Microservices architecture
- ✅ Comprehensive analytics
- ✅ Clean code principles
- ✅ Docker containerization
- ✅ Scalable system design

---

**⭐ Star this repo if you find it helpful!**
