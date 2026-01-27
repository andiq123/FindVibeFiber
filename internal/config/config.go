package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/core/constants"
	"github.com/andiq123/FindVibeFiber/internal/utils"
)

type DatabaseConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

type HTTPConfig struct {
	Timeout        time.Duration
	MaxIdleConns   int
	MaxIdlePerHost int
	IdleTimeout    time.Duration
}

type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type SearchConfig struct {
	Timeout      time.Duration
	MaxResults   int
	MaxPage      int
	MaxQueryLen  int
	MinQueryLen  int
}

type AppConfig struct {
	Database DatabaseConfig
	HTTP     HTTPConfig
	Server   ServerConfig
	Search   SearchConfig
}

func LoadDatabaseConfig() DatabaseConfig {
	port, err := strconv.Atoi(utils.GetEnvOrDef("DB_PORT", fmt.Sprintf("%d", constants.DefaultDBPort)))
	if err != nil {
		utils.GetLogger().Warn("Invalid DB_PORT, using default", "default", constants.DefaultDBPort, "error", err)
		port = constants.DefaultDBPort
	}

	maxOpenConns := parseIntEnv("DB_MAX_OPEN_CONNS", constants.DefaultDBMaxOpenConns)
	maxIdleConns := parseIntEnv("DB_MAX_IDLE_CONNS", constants.DefaultDBMaxIdleConns)
	connMaxLifetime := time.Duration(parseIntEnv("DB_CONN_MAX_LIFETIME_MIN", constants.DefaultDBConnMaxLifetime)) * time.Minute
	connMaxIdleTime := time.Duration(parseIntEnv("DB_CONN_MAX_IDLE_TIME_MIN", constants.DefaultDBConnMaxIdleTime)) * time.Minute

	return DatabaseConfig{
		Host:            utils.GetEnvOrDef("DB_HOST", "localhost"),
		Port:            port,
		User:            utils.GetEnvOrDef("DB_USER", "postgres"),
		Password:        utils.GetEnvOrDef("DB_PASSWORD", "postgres"),
		DBName:          utils.GetEnvOrDef("DB_NAME", "findvibe"),
		SSLMode:         utils.GetEnvOrDef("DB_SSLMODE", "disable"),
		MaxOpenConns:    maxOpenConns,
		MaxIdleConns:    maxIdleConns,
		ConnMaxLifetime: connMaxLifetime,
		ConnMaxIdleTime: connMaxIdleTime,
	}
}

func LoadHTTPConfig() HTTPConfig {
	timeout := time.Duration(parseIntEnv("HTTP_TIMEOUT_SEC", constants.DefaultHTTPTimeout)) * time.Second
	maxIdleConns := parseIntEnv("HTTP_MAX_IDLE_CONNS", constants.DefaultHTTPMaxIdleConns)
	maxIdlePerHost := parseIntEnv("HTTP_MAX_IDLE_PER_HOST", constants.DefaultHTTPMaxIdlePerHost)
	idleTimeout := time.Duration(parseIntEnv("HTTP_IDLE_TIMEOUT_SEC", constants.DefaultHTTPIdleTimeout)) * time.Second

	return HTTPConfig{
		Timeout:        timeout,
		MaxIdleConns:   maxIdleConns,
		MaxIdlePerHost: maxIdlePerHost,
		IdleTimeout:    idleTimeout,
	}
}

func LoadServerConfig() ServerConfig {
	port := utils.GetEnvOrDef("PORT", constants.DefaultServerPort)
	readTimeout := time.Duration(parseIntEnv("SERVER_READ_TIMEOUT_SEC", constants.DefaultReadTimeout)) * time.Second
	writeTimeout := time.Duration(parseIntEnv("SERVER_WRITE_TIMEOUT_SEC", constants.DefaultWriteTimeout)) * time.Second
	idleTimeout := time.Duration(parseIntEnv("SERVER_IDLE_TIMEOUT_SEC", constants.DefaultIdleTimeout)) * time.Second

	return ServerConfig{
		Port:         port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}
}

func LoadSearchConfig() SearchConfig {
	timeout := time.Duration(parseIntEnv("SEARCH_TIMEOUT_SEC", constants.DefaultSearchTimeout)) * time.Second
	maxResults := parseIntEnv("SEARCH_MAX_RESULTS", constants.DefaultMaxSearchResults)
	maxPage := parseIntEnv("SEARCH_MAX_PAGE", constants.DefaultMaxPageNumber)
	maxQueryLen := parseIntEnv("SEARCH_MAX_QUERY_LEN", constants.MaxQueryLength)
	minQueryLen := parseIntEnv("SEARCH_MIN_QUERY_LEN", constants.MinQueryLength)

	return SearchConfig{
		Timeout:     timeout,
		MaxResults:  maxResults,
		MaxPage:     maxPage,
		MaxQueryLen: maxQueryLen,
		MinQueryLen: minQueryLen,
	}
}

func LoadConfig() *AppConfig {
	return &AppConfig{
		Database: LoadDatabaseConfig(),
		HTTP:     LoadHTTPConfig(),
		Server:   LoadServerConfig(),
		Search:   LoadSearchConfig(),
	}
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		c.Host, c.Port, c.User, c.Password, c.DBName, c.SSLMode)
}

func parseIntEnv(key string, defaultValue int) int {
	value := utils.GetEnvOrDef(key, "")
	if value == "" {
		return defaultValue
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		utils.GetLogger().Warn("Invalid environment variable, using default", "key", key, "default", defaultValue, "error", err)
		return defaultValue
	}
	return parsed
}
