package config

import (
	"fmt"
	"log"
	"strconv"

	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func LoadDatabaseConfig() DatabaseConfig {
	port, err := strconv.Atoi(utils.GetEnvOrDef("DB_PORT", "5432"))
	if err != nil {
		log.Printf("Invalid DB_PORT, using default 5432: %v", err)
		port = 5432
	}

	return DatabaseConfig{
		Host:     utils.GetEnvOrDef("DB_HOST", "localhost"),
		Port:     port,
		User:     utils.GetEnvOrDef("DB_USER", "postgres"),
		Password: utils.GetEnvOrDef("DB_PASSWORD", "postgres"),
		DBName:   utils.GetEnvOrDef("DB_NAME", "findvibe"),
		SSLMode:  utils.GetEnvOrDef("DB_SSLMODE", "disable"),
	}
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}
