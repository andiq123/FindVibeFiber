package utils

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func GetEnvOrDef(key, def string) string {
	value := os.Getenv(key)

	if value == "" {
		return def
	}

	return value
}

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}
}
