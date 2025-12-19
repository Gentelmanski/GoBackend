package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"student-backend/auth"
	"student-backend/middleware"
	"student-backend/models"

	"gorm.io/gorm"
)

type AuthHandler struct {
	db         *gorm.DB
	jwtService *auth.JWTService
}

func NewAuthHandler(db *gorm.DB, jwtService *auth.JWTService) *AuthHandler {
	return &AuthHandler{
		db:         db,
		jwtService: jwtService,
	}
}

// Login обрабатывает вход пользователя
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var loginReq models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&loginReq); err != nil {
		log.Printf("❌ Error decoding login request: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Ищем пользователя
	var user models.User
	result := h.db.Where("email = ?", loginReq.Email).First(&user)
	if result.Error != nil {
		log.Printf("❌ User not found: %s", loginReq.Email)
		http.Error(w, `{"error": "Invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	// Проверяем пароль
	if !auth.CheckPassword(loginReq.Password, user.Password) {
		log.Printf("❌ Invalid password for user: %s", loginReq.Email)
		http.Error(w, `{"error": "Invalid email or password"}`, http.StatusUnauthorized)
		return
	}

	// Генерируем токен
	token, err := h.jwtService.GenerateToken(&user)
	if err != nil {
		log.Printf("❌ Error generating token for user %s: %v", user.Email, err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Скрываем пароль в ответе
	user.Password = ""

	response := models.LoginResponse{
		Token: token,
		User:  user,
	}

	log.Printf("✅ User logged in successfully: %s (role: %s)", user.Email, user.Role)
	json.NewEncoder(w).Encode(response)
}

// Register регистрирует нового пользователя
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var registerReq models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&registerReq); err != nil {
		log.Printf("❌ Error decoding register request: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли пользователь
	var existingUser models.User
	if err := h.db.Where("email = ?", registerReq.Email).First(&existingUser).Error; err == nil {
		log.Printf("❌ User already exists: %s", registerReq.Email)
		http.Error(w, `{"error": "User with this email already exists"}`, http.StatusConflict)
		return
	}

	// Хэшируем пароль
	hashedPassword, err := auth.HashPassword(registerReq.Password)
	if err != nil {
		log.Printf("❌ Error hashing password: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Создаем пользователя
	user := models.User{
		Email:    registerReq.Email,
		Password: hashedPassword,
		Role:     registerReq.Role,
	}

	// Создаем связанные записи в зависимости от роли
	switch registerReq.Role {
	case models.RoleStudent:
		// Создаем студента
		student := models.Student{
			Email:   registerReq.Email,
			Name:    "New",
			Surname: "Student",
		}
		if err := h.db.Create(&student).Error; err != nil {
			log.Printf("❌ Error creating student: %v", err)
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
			return
		}
		user.StudentID = &student.ID

	case models.RoleTeacher:
		// Создаем преподавателя
		teacher := models.Teacher{
			Email:   registerReq.Email,
			Name:    "New",
			Surname: "Teacher",
		}
		if err := h.db.Create(&teacher).Error; err != nil {
			log.Printf("❌ Error creating teacher: %v", err)
			http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
			return
		}
		user.TeacherID = &teacher.ID
	}

	// Сохраняем пользователя
	if err := h.db.Create(&user).Error; err != nil {
		log.Printf("❌ Error creating user: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Обновляем связанные записи
	switch registerReq.Role {
	case models.RoleStudent:
		h.db.Model(&models.Student{ID: *user.StudentID}).Update("user_id", user.ID)
	case models.RoleTeacher:
		h.db.Model(&models.Teacher{ID: *user.TeacherID}).Update("user_id", user.ID)
	}

	// Генерируем токен
	token, err := h.jwtService.GenerateToken(&user)
	if err != nil {
		log.Printf("❌ Error generating token: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Скрываем пароль в ответе
	user.Password = ""

	response := models.LoginResponse{
		Token: token,
		User:  user,
	}

	log.Printf("✅ User registered successfully: %s (role: %s)", user.Email, user.Role)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// GetCurrentUser возвращает текущего пользователя
func (h *AuthHandler) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Извлекаем claims из контекста (через middleware)
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	// Получаем полную информацию о пользователе
	var user models.User
	if err := h.db.Preload("Student").Preload("Teacher").First(&user, claims.UserID).Error; err != nil {
		log.Printf("❌ Error fetching user: %v", err)
		http.Error(w, `{"error": "User not found"}`, http.StatusNotFound)
		return
	}

	// Скрываем пароль
	user.Password = ""
	json.NewEncoder(w).Encode(user)
}
