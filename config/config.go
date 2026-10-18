package config

import (
	"log"
	"os"
	"strconv"
	"strings"

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
	AllowedOrigins       []string

	STTEnabled        bool
	STTURL            string
	STTAPIKey         string
	STTModel          string
	STTLanguage       string
	STTPrompt         string
	STTBeamSize       int
	STTTimeoutSeconds int

	LLMEnabled        bool
	LLMURL            string
	LLMAPIKey         string
	LLMModel          string
	LLMTimeoutSeconds int
}

const defaultSTTPrompt = "Catatan keuangan berbahasa Indonesia. Contoh nominal: sepuluh ribu, lima puluh ribu, seratus ribu, lima juta."

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("Warning: .env file not found, using environment variables")
	}

	cfg := &Config{
		Port:                 getEnv("PORT", "8080"),
		GoogleClientID:       getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret:   getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:    getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/auth/google/callback"),
		GoogleServiceAccFile: getEnv("GOOGLE_SERVICE_ACCOUNT_FILE", "service_account.json"),
		JWTSecret:            getEnv("JWT_SECRET", "secret"),
		FrontendURL:          getEnv("FRONTEND_URL", "http://localhost:3000"),
		AllowedOrigins:       strings.Split(getEnv("ALLOWED_ORIGINS", "http://localhost:3000"), ","),

		STTEnabled:        getEnvBool("STT_ENABLED", false),
		STTURL:            getEnv("STT_URL", "http://localhost:8081/inference"),
		STTAPIKey:         getEnv("STT_API_KEY", ""),
		STTModel:          getEnv("STT_MODEL", ""),
		STTLanguage:       getEnv("STT_LANGUAGE", "id"),
		STTPrompt:         getEnv("STT_PROMPT", defaultSTTPrompt),
		STTBeamSize:       getEnvInt("STT_BEAM_SIZE", 0),
		STTTimeoutSeconds: getEnvInt("STT_TIMEOUT_SECONDS", 30),

		LLMEnabled:        getEnvBool("LLM_ENABLED", false),
		LLMURL:            getEnv("LLM_URL", ""),
		LLMAPIKey:         getEnv("LLM_API_KEY", ""),
		LLMModel:          getEnv("LLM_MODEL", ""),
		LLMTimeoutSeconds: getEnvInt("LLM_TIMEOUT_SECONDS", 15),
	}

	if cfg.LLMAPIKey == "" {
		cfg.LLMAPIKey = cfg.STTAPIKey
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok {
		return fallback
	}
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return fallback
	}
	return parsed
}
