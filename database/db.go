package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // драйвер PostgreSQL
)

const (
	host     = "localhost"
	port     = 5432
	user     = "max"
	password = "12345"
	dbname   = "students_db"
)

func InitDB() (*sqlx.DB, error) {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	// Сначала используем стандартный database/sql
	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Затем оборачиваем в sqlx
	dbx := sqlx.NewDb(db, "postgres")

	// Проверяем подключение
	err = dbx.Ping()
	if err != nil {
		return nil, fmt.Errorf("error pinging database: %v", err)
	}

	// Создаем таблицу
	err = createTable(db)
	if err != nil {
		return nil, fmt.Errorf("error creating table: %v", err)
	}

	log.Println("Successfully connected to PostgreSQL!")
	return dbx, nil
}

func createTable(db *sql.DB) error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS students (
        id SERIAL PRIMARY KEY,
        name VARCHAR(100) NOT NULL,
        surname VARCHAR(100) NOT NULL
    );`

	_, err := db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("error creating students table: %v", err)
	}

	// Исправляем последовательность, если она сбилась
	err = fixSequence(db)
	if err != nil {
		log.Printf("⚠️ Warning: could not fix sequence: %v", err)
	}

	log.Println("✅ Students table verified (id, name, surname)")
	return nil
}

func fixSequence(db *sql.DB) error {
	// Устанавливаем правильное значение для последовательности
	fixSeqSQL := `
    SELECT setval(
        'students_id_seq',
        COALESCE((SELECT MAX(id) FROM students), 0) + 1,
        false
    )`

	var result int64
	err := db.QueryRow(fixSeqSQL).Scan(&result)
	if err != nil {
		return fmt.Errorf("error fixing sequence: %v", err)
	}

	log.Printf("✅ Sequence fixed, next ID will be: %d", result)
	return nil
}
