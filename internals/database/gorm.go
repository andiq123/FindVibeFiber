package database

import (
	"log"
	"net/url"
	"time"

	"github.com/andiq123/FindVibeFiber/internals/core/domain"
	"github.com/andiq123/FindVibeFiber/internals/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDb() *gorm.DB {
	serviceURI := utils.GetEnvOrDef("POSTGRES_URI", "sqlite://findVibe.db")

	conn, _ := url.Parse(serviceURI)
	conn.RawQuery = "sslmode=verify-ca;sslrootcert=ca.pem"

	db, err := gorm.Open(postgres.Open(conn.String()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		log.Println("failed to connect database")
		log.Fatal(err.Error())
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("failed to get sql.DB: ", err.Error())
	}

	sqlDB.SetMaxOpenConns(10)           // Limit the number of open connections
	sqlDB.SetMaxIdleConns(5)            // Limit idle connections
	sqlDB.SetConnMaxLifetime(time.Hour) // Set maximum connection lifetime

	db.Exec("SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid <> pg_backend_pid() AND datname = current_database();")

	db.AutoMigrate(&domain.User{})
	db.AutoMigrate(&domain.FavoriteSong{})

	return db
}
