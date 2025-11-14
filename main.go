package main

import (
	"encoding/json"
	"log"
	"net/http"
	"student-backend/database"
	"student-backend/handlers"
	"student-backend/middleware"
	"time"

	"github.com/gorilla/mux"
)

func main() {
	log.Println("🚀 Starting Student Backend Server with GORM...")

	// Инициализация подключения к базе данных
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("❌ Error initializing database:", err)
	}

	// Инициализация обработчиков
	studentHandler := handlers.NewStudentHandler(db)

	// Создание роутера
	r := mux.NewRouter()

	// Добавление CORS middleware
	r.Use(middleware.CORS)

	// Логирование всех запросов
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Printf("📨 %s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

	// Корневой маршрут
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Student Backend API</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background: #f5f5f5; }
        .container { background: white; padding: 20px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        .status { background: #4CAF50; color: white; padding: 10px; border-radius: 5px; display: inline-block; }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎓 Student Backend API</h1>
        <div class="status">✅ Сервер работает корректно</div>
        <p>ORM: GORM | Database: PostgreSQL</p>
    </div>
</body>
</html>`
		w.Write([]byte(html))
	})

	// Маршруты API
	api := r.PathPrefix("/api").Subrouter()

	api.HandleFunc("/students", studentHandler.GetStudents).Methods("GET")
	api.HandleFunc("/students", studentHandler.CreateStudent).Methods("POST")
	api.HandleFunc("/students/{id}", studentHandler.UpdateStudent).Methods("PATCH", "PUT")
	api.HandleFunc("/students/{id}", studentHandler.DeleteStudent).Methods("DELETE")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		sqlDB, err := db.DB()
		dbStatus := "connected"
		if err != nil {
			dbStatus = "error"
		} else {
			if err := sqlDB.Ping(); err != nil {
				dbStatus = "disconnected"
			}
		}

		response := map[string]interface{}{
			"status":    "ok",
			"service":   "student-backend",
			"orm":       "GORM",
			"database":  dbStatus,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// OPTIONS handlers
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	log.Println("✅ Server successfully started on :8080")
	log.Println("🌐 Available at: http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}
