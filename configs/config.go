package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

// Config holds the application configuration loaded from environment variables.
type Config struct {
	BaseURL string
	DBPath  string
}

// Load reads environment variables and returns a validated Config.
func Load() (*Config, error) {
	_ = godotenv.Load()

	baseURL := os.Getenv("APP_BASE_URL")
	if baseURL == "" {
		return nil, fmt.Errorf("APP_BASE_URL environment variable is not set")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "internal/database/data.db"
	}

	return &Config{BaseURL: baseURL, DBPath: dbPath}, nil
}
