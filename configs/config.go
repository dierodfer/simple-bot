package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var BaseURL string

func InitVars() {
	_ = godotenv.Load()

	BaseURL = os.Getenv("APP_BASE_URL")
	if BaseURL == "" {
		log.Fatal("APP_BASE_URL not set in .env file")
	}
}
