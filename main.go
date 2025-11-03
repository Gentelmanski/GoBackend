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
	_ "github.com/lib/pq"
)

func main() {
	log.Println("🚀 Starting Student Backend Server...")

	// Инициализация подключения к базе данных
	db, err := database.InitDB()
	if err != nil {
		log.Fatal("❌ Error initializing database:", err)
	}
	defer db.Close()

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

	// Корневой маршрут - информация о сервере
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Student Backend API</title>
    <style>
        body {
            font-family: 'Arial', sans-serif;
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
            margin-bottom: 2rem;
        }
        .endpoints {
            text-align: left;
            background: #f8f9fa;
            padding: 1.5rem;
            border-radius: 10px;
            margin: 1.5rem 0;
        }
        .endpoint {
            margin: 0.5rem 0;
            font-family: 'Courier New', monospace;
            padding: 0.3rem 0.6rem;
            background: #e9ecef;
            border-radius: 5px;
        }
        .footer {
            margin-top: 2rem;
            color: #666;
            font-size: 0.9rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎓 Student Backend API</h1>
        <div class="status">
            ✅ Сервер работает корректно
        </div>

        <div class="endpoints">
            <h3>📡 Доступные endpoints:</h3>
            <div class="endpoint"><strong>GET</strong> /api/students - Получить список студентов</div>
            <div class="endpoint"><strong>POST</strong> /api/students - Добавить нового студента</div>
            <div class="endpoint"><strong>PATCH</strong> /api/students/{id} - Обновить студента</div>
            <div class="endpoint"><strong>DELETE</strong> /api/students/{id} - Удалить студента</div>
            <div class="endpoint"><strong>GET</strong> /health - Проверка здоровья сервера</div>
        </div>

        <div class="footer">
            <p>Backend: Go + PostgreSQL | Frontend: Angular</p>
            <p>🕒 Сервер запущен и готов к работе</p>
        </div>
    </div>
</body>
</html>`
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(html))
	})

	// Маршруты API
	api := r.PathPrefix("/api").Subrouter()

	// Студенты
	api.HandleFunc("/students", studentHandler.GetStudents).Methods("GET")
	api.HandleFunc("/students", studentHandler.CreateStudent).Methods("POST")
	api.HandleFunc("/students", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("OPTIONS")

	api.HandleFunc("/students/{id}", studentHandler.UpdateStudent).Methods("PATCH", "PUT")
	api.HandleFunc("/students/{id}", studentHandler.DeleteStudent).Methods("DELETE")
	api.HandleFunc("/students/{id}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}).Methods("OPTIONS")

	// Health check
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Проверяем подключение к базе данных
		err := db.Ping()
		dbStatus := "connected"
		if err != nil {
			dbStatus = "disconnected"
			log.Printf("❌ Health check: Database ping failed: %v", err)
		}

		response := map[string]interface{}{
			"status":    "ok",
			"service":   "student-backend",
			"database":  dbStatus,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// Универсальный обработчик OPTIONS для всех путей
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔄 Global OPTIONS handler for: %s", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})

	log.Println("✅ Server successfully started on :8080")
	log.Println("🌐 Available at: http://localhost:8080")
	log.Println("📋 API endpoints:")
	log.Println("   GET    /api/students")
	log.Println("   POST   /api/students")
	log.Println("   PATCH  /api/students/{id}")
	log.Println("   DELETE /api/students/{id}")
	log.Println("   GET    /health")

	log.Fatal(http.ListenAndServe(":8080", r))
}
