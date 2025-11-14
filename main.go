package main

import (
	"encoding/json"
	"log"
	"net/http"
	"student-backend/config"
	"student-backend/database"
	"student-backend/handlers"
	"student-backend/middleware"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func main() {
	log.Println("🚀 Starting Student Backend Server with GORM...")

	// Загрузка конфигурации
	cfg := config.Load()
	log.Printf("📋 Configuration loaded: Server Port %s", cfg.ServerPort)

	// Инициализация подключения к базе данных
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal("❌ Error initializing database:", err)
	}

	// Получаем низкоуровневое соединение для закрытия
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("❌ Error getting SQL DB:", err)
	}
	defer sqlDB.Close()

	// Инициализация обработчиков
	studentHandler := handlers.NewStudentHandler(db)

	// Создание роутера
	r := mux.NewRouter()

	// Добавление middleware
	r.Use(middleware.CORS)
	r.Use(loggingMiddleware)

	// Маршруты
	setupRoutes(r, studentHandler, db)

	serverAddr := ":" + cfg.ServerPort
	log.Printf("✅ Server successfully started on %s", serverAddr)
	log.Printf("🌐 Available at: http://localhost%s", serverAddr)

	log.Fatal(http.ListenAndServe(serverAddr, r))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("📨 %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func setupRoutes(r *mux.Router, studentHandler *handlers.StudentHandler, db *gorm.DB) {
	// Корневой маршрут
	r.HandleFunc("/", rootHandler).Methods("GET")

	// API маршруты
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/students", studentHandler.GetStudents).Methods("GET")
	api.HandleFunc("/students", studentHandler.CreateStudent).Methods("POST")
	api.HandleFunc("/students/{id}", studentHandler.UpdateStudent).Methods("PATCH", "PUT")
	api.HandleFunc("/students/{id}", studentHandler.DeleteStudent).Methods("DELETE")

	// Health check
	r.HandleFunc("/health", healthHandler(db)).Methods("GET")

	// OPTIONS handlers
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := `
<!DOCTYPE html>
<html>
<head>
    <title>Student Backend API</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 0;
            padding: 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .container {
            background: white;
            padding: 3rem;
            border-radius: 15px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            text-align: center;
            max-width: 600px;
        }
        h1 {
            color: #333;
            margin-bottom: 1.5rem;
        }
        .status {
            background: #4CAF50;
            color: white;
            padding: 0.5rem 1rem;
            border-radius: 25px;
            display: inline-block;
            margin-bottom: 1rem;
        }
        .tech {
            background: #f8f9fa;
            padding: 1rem;
            border-radius: 10px;
            margin: 1rem 0;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎓 Student Backend API</h1>
        <div class="status">✅ Сервер работает корректно</div>
        <div class="tech">
            <p><strong>ORM:</strong> GORM</p>
            <p><strong>Database:</strong> PostgreSQL</p>
            <p><strong>Framework:</strong> Gorilla Mux</p>
        </div>
        <p>API endpoints available at <code>/api/students</code></p>
    </div>
</body>
</html>`
	w.Write([]byte(html))
}

func healthHandler(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		sqlDB, err := db.DB()
		dbStatus := "connected"
		if err != nil {
			dbStatus = "error"
		} else {
			if err := sqlDB.Ping(); err != nil {
				dbStatus = "disconnected"
				log.Printf("❌ Database ping failed: %v", err)
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
	}
}
