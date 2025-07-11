package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBUrl      string
	ServerPort string
	Env        string
}

// LoadConfig loads configuration from environment variables.
// Returns an error if required environment variables are missing.
func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	// Validate required environment variables
	requiredVars := map[string]string{
		"POSTGRES_HOST":     os.Getenv("POSTGRES_HOST"),
		"POSTGRES_PORT":     os.Getenv("POSTGRES_PORT"),
		"POSTGRES_USER":     os.Getenv("POSTGRES_USER"),
		"POSTGRES_PASSWORD": os.Getenv("POSTGRES_PASSWORD"),
		"POSTGRES_DB":       os.Getenv("POSTGRES_DB"),
	}

	// Check for missing required variables
	var missingVars []string
	for name, value := range requiredVars {
		if value == "" {
			missingVars = append(missingVars, name)
		}
	}

	if len(missingVars) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingVars)
	}

	// Build database connection string
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		requiredVars["POSTGRES_HOST"],
		requiredVars["POSTGRES_PORT"],
		requiredVars["POSTGRES_USER"],
		requiredVars["POSTGRES_PASSWORD"],
		requiredVars["POSTGRES_DB"],
	)

	cfg := &Config{
		DBUrl:      dsn,
		ServerPort: os.Getenv("SERVER_PORT"),
		Env:        os.Getenv("APP_ENV"),
	}

	// Set defaults
	if cfg.ServerPort == "" {
		cfg.ServerPort = "3000"
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	return cfg, nil
}
