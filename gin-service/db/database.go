package db

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect() {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Kolkata",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USERNAME", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "banking"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSLMODE", "disable"),
	)

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	DB = database

	createSQL := `
		CREATE TABLE IF NOT EXISTS transfers (
			id            UUID        PRIMARY KEY,
			from_account  UUID        NOT NULL,
			to_account    UUID        NOT NULL,
			amount        NUMERIC     NOT NULL,
			transfer_mode VARCHAR(20) NOT NULL,
			status        VARCHAR(20) NOT NULL,
			created_at    TIMESTAMPTZ DEFAULT now()
		);
	`
	if err := DB.Exec(createSQL).Error; err != nil {
		log.Fatal("Failed to migrate database:", err)
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}