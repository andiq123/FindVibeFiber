package database

import (
	"log"
	"net/url"

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
		log.Fatal("failed to connect database")
	}

	db.AutoMigrate(&domain.User{})
	db.AutoMigrate(&domain.FavoriteSong{})

	return db
}
