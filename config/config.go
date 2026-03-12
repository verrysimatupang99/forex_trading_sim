package config

import (
	"errors"
	"os"
	"strconv"
)

// Config holds all configuration for the application
type Config struct {
	// Server
	ServerPort string

	// Database
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string

	// Redis
	RedisHost string
	RedisPort string

	// JWT
	JWTSecret          string
	JWTExpirationHours int

	// External APIs
	AlphaVantageAPIKey string
	OANDAAPIKey        string
	OANDAAccountID     string

	// ML Model
	ModelPath string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Validate required environment variables
	err := validateRequired()
	if err != nil {
		return nil, err
	}

	return &Config{
		ServerPort:          getEnv("SERVER_PORT", "8080"),
		DBHost:              getEnv("DB_HOST", "localhost"),
		DBPort:              getEnv("DB_PORT", "5432"),
		DBUser:              getEnv("DB_USER", "postgres"),
		DBPassword:          getEnv("DB_PASSWORD", "postgres"),
		DBName:              getEnv("DB_NAME", "forex_sim"),
		RedisHost:           getEnv("REDIS_HOST", "localhost"),
		RedisPort:           getEnv("REDIS_PORT", "6379"),
		JWTSecret:           getEnv("JWT_SECRET", "your-secret-key-change-in-production"),
		JWTExpirationHours:  getEnvInt("JWT_EXPIRATION_HOURS", 24),
		AlphaVantageAPIKey:  getEnv("ALPHA_VANTAGE_API_KEY", ""),
		OANDAAPIKey:         getEnv("OANDA_API_KEY", ""),
		OANDAAccountID:      getEnv("OANDA_ACCOUNT_ID", ""),
		ModelPath:           getEnv("MODEL_PATH", "./models"),
	}, nil
}

// validateRequired checks for required environment variables
func validateRequired() error {
	required := []string{"DB_HOST", "DB_USER", "DB_PASSWORD", "DB_NAME"}

	for _, key := range required {
		if value := os.Getenv(key); value == "" {
			return errors.New("required environment variable not set: " + key)
		}
	}

	// Warn about JWT secret in production
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" || jwtSecret == "your-secret-key-change-in-production" {
		return errors.New("JWT_SECRET must be set to a secure value in production")
	}

	return nil
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
