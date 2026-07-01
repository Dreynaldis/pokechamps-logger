package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL    string
	Port           string
	AuthSecret     string
	FrontendOrigin string
}

func Load() *Config {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, reading from environment")
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	return &Config{
		DatabaseURL:    requireEnv("DATABASE_URL"),
		Port:           port,
		AuthSecret:     os.Getenv("AUTH_SECRET"),
		FrontendOrigin: os.Getenv("FRONTEND_ORIGIN"),
	}
}

func requireEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required environment variable %s is not set", key)
	}
	return v
}
