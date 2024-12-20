package utils

import (
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

func LoadEnv() error {
	return godotenv.Load()
}

func IsDebug() bool {
	return GetEnvOrDef("DEBUG", "true") == "true"
}
