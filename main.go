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
	teacherHandler := handlers.NewTeacherHandler(db)

	// Создание роутера
	r := mux.NewRouter()

	// Добавление middleware CORS для всех маршрутов
	r.Use(middleware.CORS)
	r.Use(loggingMiddleware)

	// Маршруты
	setupRoutes(r, authHandler, studentHandler, teacherHandler, authMiddleware)

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
	studentHandler *handlers.StudentHandler,
	teacherHandler *handlers.TeacherHandler,
	authMiddleware *middleware.AuthMiddleware) {

	// Создаем отдельный роутер для API с middleware аутентификации
	api := r.PathPrefix("/api").Subrouter()

	// Публичные маршруты API (без аутентификации)
	api.HandleFunc("/auth/login", authHandler.Login).Methods("POST")
	api.HandleFunc("/auth/register", authHandler.Register).Methods("POST")

	// Защищенные маршруты API
	protectedAPI := r.PathPrefix("/api").Subrouter()
	protectedAPI.Use(authMiddleware.AuthMiddleware)

	// Аутентификация
	protectedAPI.HandleFunc("/auth/me", authHandler.GetCurrentUser).Methods("GET")

	// Студенты
	protectedAPI.HandleFunc("/students", studentHandler.GetStudents).Methods("GET")
	protectedAPI.HandleFunc("/students", studentHandler.CreateStudent).Methods("POST")
	protectedAPI.HandleFunc("/students/{id}", studentHandler.UpdateStudent).Methods("PUT", "PATCH")
	protectedAPI.HandleFunc("/students/{id}", studentHandler.DeleteStudent).Methods("DELETE")

	// Преподаватели - ТОЛЬКО для админа
	protectedAPI.HandleFunc("/teachers", teacherHandler.GetTeachers).Methods("GET")
	protectedAPI.HandleFunc("/teachers", teacherHandler.CreateTeacher).Methods("POST")
	protectedAPI.HandleFunc("/teachers/{id}", teacherHandler.UpdateTeacher).Methods("PUT", "PATCH")
	protectedAPI.HandleFunc("/teachers/{id}", teacherHandler.DeleteTeacher).Methods("DELETE")

	// Публичные маршруты (без API префикса)
	r.HandleFunc("/", rootHandler).Methods("GET")
	r.HandleFunc("/health", healthHandler).Methods("GET")

	// OPTIONS handlers для всех маршрутов
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With, Accept, Origin")
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
                <li><code>GET /api/teachers</code> - Get teachers (Admin only)</li>
                <li><code>POST /api/teachers</code> - Create teacher (Admin only)</li>
                <li><code>PUT/PATCH /api/teachers/{id}</code> - Update teacher (Admin only)</li>
                <li><code>DELETE /api/teachers/{id}</code> - Delete teacher (Admin only)</li>
            </ul>
        </div>
        <p>Default admin: <code>admin@example.com</code> / <code>admin123</code></p>
    </div>
</body>
</html>`
	w.Write([]byte(html))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := map[string]interface{}{
		"status":    "ok",
		"service":   "student-backend",
		"orm":       "GORM",
		"auth":      "JWT",
		"timestamp": time.Now().Format(time.RFC3339),
	}

	json.NewEncoder(w).Encode(response)
}
