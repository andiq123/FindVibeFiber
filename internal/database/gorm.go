package database

import (
	"log"
	"time"

	"github.com/andiq123/FindVibeFiber/internal/config"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDb() *gorm.DB {
	var db *gorm.DB
	var err error

	dbConfig := config.LoadDatabaseConfig()
	dsn := dbConfig.DSN()

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database connection: %v", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)
	sqlDB.SetConnMaxIdleTime(10 * time.Minute)

	log.Println("PostgreSQL database connected successfully")

	runMigrations(db)

	return db
}

func runMigrations(db *gorm.DB) {
	migrations := []string{
		`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
		`CREATE EXTENSION IF NOT EXISTS "pg_stat_statements"`,
		`CREATE TABLE IF NOT EXISTS users (
			id VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL UNIQUE
		)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_users_name ON users(name)`,
		`CREATE TABLE IF NOT EXISTS favorite_songs (
			id VARCHAR(255) PRIMARY KEY,
			title VARCHAR(500) NOT NULL,
			artist VARCHAR(500) NOT NULL,
			image VARCHAR(1000),
			link VARCHAR(1000),
			"order" INTEGER NOT NULL DEFAULT 0,
			user_uuid VARCHAR(255) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
			CONSTRAINT fk_favorite_songs_user FOREIGN KEY (user_uuid) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_songs_id_user ON favorite_songs(id, user_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_songs_user_order ON favorite_songs(user_uuid, "order")`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_songs_user_uuid ON favorite_songs(user_uuid)`,
		`CREATE INDEX IF NOT EXISTS idx_favorite_songs_created_at ON favorite_songs(created_at)`,
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = CURRENT_TIMESTAMP;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		`DO $$
		BEGIN
			IF NOT EXISTS (
				SELECT 1 FROM pg_trigger WHERE tgname = 'update_favorite_songs_updated_at'
			) THEN
				CREATE TRIGGER update_favorite_songs_updated_at
					BEFORE UPDATE ON favorite_songs
					FOR EACH ROW
					EXECUTE FUNCTION update_updated_at_column();
			END IF;
		END $$`,
	}

	for _, migration := range migrations {
		if err := db.Exec(migration).Error; err != nil {
			log.Printf("Migration warning: %v", err)
		}
	}

	log.Println("Database migrations completed")
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
