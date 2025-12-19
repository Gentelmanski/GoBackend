package main

import (
	"encoding/json"
	"log"
	"net/http"
	"student-backend/auth"
	"student-backend/config"
	"student-backend/database"
	"student-backend/handlers"
	"student-backend/middleware"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func main() {
	log.Println("🚀 Starting Student Backend Server with Authentication...")

	// Загрузка конфигурации
	cfg := config.Load()
	log.Printf("📋 Configuration loaded: Server Port %s", cfg.ServerPort)

	// Инициализация подключения к базе данных
	db, err := database.InitDB(cfg)
	if err != nil {
		log.Fatal("❌ Error initializing database:", err)
	}

	// Выполняем миграции
	if err := database.Migrate(db); err != nil {
		log.Fatal("❌ Error migrating database:", err)
	}

	// Получаем низкоуровневое соединение для закрытия
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("❌ Error getting SQL DB:", err)
	}
	defer sqlDB.Close()

	// Инициализация JWT сервиса
	jwtService := auth.NewJWTService(cfg.JWTSecret, cfg.JWTExpiry)

	// Инициализация middleware
	authMiddleware := middleware.NewAuthMiddleware(jwtService)

	// Инициализация обработчиков
	authHandler := handlers.NewAuthHandler(db, jwtService)
	studentHandler := handlers.NewStudentHandler(db)

	// Создание роутера
	r := mux.NewRouter()

	// Добавление middleware
	r.Use(middleware.CORS)
	r.Use(loggingMiddleware)

	// Маршруты
	setupRoutes(r, authHandler, studentHandler, db, authMiddleware)

	serverAddr := ":" + cfg.ServerPort
	log.Printf("✅ Server successfully started on %s", serverAddr)
	log.Printf("🌐 Available at: http://localhost%s", serverAddr)
	log.Printf("🔐 JWT Expiry: %d hours", cfg.JWTExpiry)

	log.Fatal(http.ListenAndServe(serverAddr, r))
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для response writer для захвата статуса
		rw := &responseWriter{w, http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("📨 %s %s - %d (%v)", r.Method, r.URL.Path, rw.statusCode, duration)
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func setupRoutes(r *mux.Router, authHandler *handlers.AuthHandler,
	studentHandler *handlers.StudentHandler, db *gorm.DB,
	authMiddleware *middleware.AuthMiddleware) {

	// Публичные маршруты (без аутентификации)
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.HandleFunc("/health", healthHandler(db)).Methods("GET")
	r.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/api/auth/register", authHandler.Register).Methods("POST")

	// Защищенные маршруты
	api := r.PathPrefix("/api").Subrouter()

	// Применяем middleware аутентификации к защищенным маршрутам
	api.Use(authMiddleware.AuthMiddleware)

	// Аутентификация
	api.HandleFunc("/auth/me", authHandler.GetCurrentUser).Methods("GET")

	// Создаем подроутеры для студентов с разными уровнями доступа

	// GET /api/students - доступен всем аутентифицированным пользователям
	studentsRouter := api.PathPrefix("/students").Subrouter()
	studentsRouter.HandleFunc("", studentHandler.GetStudents).Methods("GET")

	// Создаем отдельный роутер для операций, требующих роли админа
	// Для этого используем отдельные обработчики или встроенную проверку в handlers

	// POST /api/students - создание студента (только админ)
	// Проверка прав будет в самом обработчике CreateStudent
	studentsRouter.HandleFunc("", studentHandler.CreateStudent).Methods("POST")

	// PUT/PATCH /api/students/{id} - обновление студента
	// Проверка прав (админ, преподаватель или студент для своих данных) в обработчике
	studentsRouter.HandleFunc("/{id}", studentHandler.UpdateStudent).Methods("PUT", "PATCH")

	// DELETE /api/students/{id} - удаление студента (только админ)
	// Проверка прав будет в самом обработчике DeleteStudent
	studentsRouter.HandleFunc("/{id}", studentHandler.DeleteStudent).Methods("DELETE")

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
        .endpoints {
            text-align: left;
            background: #f1f3f4;
            padding: 1rem;
            border-radius: 8px;
            margin-top: 1rem;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎓 Student Backend API with Authentication</h1>
        <div class="status">✅ Сервер работает корректно</div>
        <div class="tech">
            <p><strong>ORM:</strong> GORM</p>
            <p><strong>Database:</strong> PostgreSQL</p>
            <p><strong>Authentication:</strong> JWT</p>
            <p><strong>Roles:</strong> Admin, Teacher, Student</p>
        </div>
        <div class="endpoints">
            <p><strong>Public Endpoints:</strong></p>
            <ul>
                <li><code>POST /api/auth/login</code> - Login</li>
                <li><code>POST /api/auth/register</code> - Register</li>
            </ul>
            <p><strong>Protected Endpoints:</strong></p>
            <ul>
                <li><code>GET /api/students</code> - Get students</li>
                <li><code>POST /api/students</code> - Create student (Admin only)</li>
                <li><code>PUT/PATCH /api/students/{id}</code> - Update student</li>
                <li><code>DELETE /api/students/{id}</code> - Delete student (Admin only)</li>
            </ul>
        </div>
        <p>Default admin: <code>admin@example.com</code> / <code>admin123</code></p>
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
			"auth":      "JWT",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		json.NewEncoder(w).Encode(response)
	}
}
