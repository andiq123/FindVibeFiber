package database

import (
	"log"
	"net/url"
	"time"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/utils"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDb() *gorm.DB {
	var db *gorm.DB
	var err error

	if utils.IsDebug() {
		db, err = gorm.Open(sqlite.Open("findVibe.db"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			log.Fatalf("Failed to connect to SQLite in debug mode: %v", err)
		}
	} else {
		serviceURI := utils.GetEnvOrDef("POSTGRES_URI", "sqlite://findVibe.db")
		conn, _ := url.Parse(serviceURI)
		q := conn.Query()
		q.Set("sslmode", "verify-ca")
		q.Set("sslrootcert", "ca.pem")
		conn.RawQuery = q.Encode()

		db, err = gorm.Open(postgres.Open(conn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			log.Fatalf("Failed to connect to PostgreSQL in release mode: %v", err)
		}
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	if err := db.AutoMigrate(&domain.User{}, &domain.FavoriteSong{}); err != nil {
		log.Fatalf("Auto migration failed: %v", err)
	}

	return db
}

func CloseDb(db *gorm.DB) {
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	if err := sqlDB.Close(); err != nil {
		log.Fatalf("Failed to close database connection: %v", err)
	}

	log.Println("Database connection closed")
}
