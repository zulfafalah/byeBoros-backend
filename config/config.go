package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port                 string
	GoogleClientID       string
	GoogleClientSecret   string
	GoogleRedirectURL    string
	GoogleServiceAccFile string
	JWTSecret            string
	FrontendURL          string
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	return &Config{
		Port:                 getEnv("PORT", "8080"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		GoogleServiceAccFile: getEnv("GOOGLE_SERVICE_ACCOUNT_FILE", "service_account.json"),
		JWTSecret:            getEnv("JWT_SECRET", "secret"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
