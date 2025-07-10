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

func LoadConfig() (*Config, error) {
	_ = godotenv.Load()

	host := os.Getenv("POSTGRES_HOST")
	if host == "" {
		panic("POSTGRES_HOST environment variable is required")
	}
	port := os.Getenv("POSTGRES_PORT")
	if port == "" {
		panic("POSTGRES_PORT environment variable is required")
	}
	user := os.Getenv("POSTGRES_USER")
	if user == "" {
		panic("POSTGRES_USER environment variable is required")
	}
	password := os.Getenv("POSTGRES_PASSWORD")
	if password == "" {
		panic("POSTGRES_PASSWORD environment variable is required")
	}
	dbname := os.Getenv("POSTGRES_DB")
	if dbname == "" {
		panic("POSTGRES_DB environment variable is required")
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	cfg := &Config{
		DBUrl:      dsn,
		ServerPort: os.Getenv("SERVER_PORT"),
		Env:        os.Getenv("APP_ENV"),
	}

	if cfg.ServerPort == "" {
		cfg.ServerPort = "3000" // default
	}
	if cfg.Env == "" {
		cfg.Env = "development"
	}

	return cfg, nil
}
