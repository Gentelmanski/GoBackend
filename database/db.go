package database

import (
	"log"
	"os"
	"student-backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() (*gorm.DB, error) {
	dsn := buildDSN()

	log.Printf("Connecting to database: %s", dsn)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Автомиграция - создаст таблицу если её нет
	err = db.AutoMigrate(&models.Student{})
	if err != nil {
		return nil, err
	}

	log.Println("✅ Successfully connected to PostgreSQL with GORM!")
	return db, nil
}

func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "max")
	password := getEnv("DB_PASSWORD", "12345")
	dbname := getEnv("DB_NAME", "students_db")
	sslmode := getEnv("DB_SSLMODE", "disable")

	return "host=" + host +
		" user=" + user +
		" password=" + password +
		" dbname=" + dbname +
		" port=" + port +
		" sslmode=" + sslmode +
		" TimeZone=UTC"
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
