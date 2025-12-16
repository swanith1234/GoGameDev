
package config

import (
	
	"os"
	"strconv"

	"github.com/joho/godotenv"
)
import "strings"


type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Game     GameConfig
	Kafka    KafkaConfig
}

type ServerConfig struct {
	Port string
	Env  string
}

// type DatabaseConfig struct {
// 	Host     string
// 	Port     string
// 	User     string
// 	Password string
// 	Name     string
// 	SSLMode  string
// }
type DatabaseConfig struct {
	DatabaseURL string
}


type GameConfig struct {
	MatchmakingTimeout  int
	ReconnectionTimeout int
}
type KafkaConfig struct {
	Brokers     []string
	TopicEvents string
	Username    string
	Password    string
}



func Load() (*Config, error) {
	_ = godotenv.Load()

	config := &Config{
		Server: ServerConfig{
			Port: getEnv("PORT", "8080"),
			Env:  getEnv("ENV", "development"),
		},
		// Database: DatabaseConfig{
		// 	Host:     getEnv("DB_HOST", "localhost"),
		// 	Port:     getEnv("DB_PORT", "5432"),
		// 	User:     getEnv("DB_USER", "postgres"),
		// 	Password: getEnv("DB_PASSWORD", "postgres"),
		// 	Name:     getEnv("DB_NAME", "connect4"),
		// 	SSLMode:  getEnv("DB_SSLMODE", "disable"),
		// },
		Database: DatabaseConfig{
	DatabaseURL: getEnv("DATABASE_URL", ""),
},

		Game: GameConfig{
			MatchmakingTimeout:  getEnvAsInt("MATCHMAKING_TIMEOUT", 10),
			ReconnectionTimeout: getEnvAsInt("RECONNECTION_TIMEOUT", 30),
		},
		Kafka: KafkaConfig{
	Brokers:     strings.Split(getEnv("KAFKA_BROKERS", ""), ","),
	TopicEvents: getEnv("KAFKA_TOPIC_EVENTS", "game.events"),
	Username:    getEnv("KAFKA_USERNAME", ""),
	Password:    getEnv("KAFKA_PASSWORD", ""),
},

	}

	return config, nil
}

// func (c *Config) GetDatabaseDSN() string {
// 	return fmt.Sprintf(
// 		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
// 		c.Database.Host,
// 		c.Database.Port,
// 		c.Database.User,
// 		c.Database.Password,
// 		c.Database.Name,
// 		c.Database.SSLMode,
// 	)
// }
func (c *Config) GetDatabaseDSN() string {
	if c.Database.DatabaseURL == "" {
		panic("DATABASE_URL is not set")
	}
	return c.Database.DatabaseURL
}


func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return defaultValue
	}
	return value
}
