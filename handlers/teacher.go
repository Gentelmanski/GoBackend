package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"student-backend/middleware"
	"student-backend/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type TeacherHandler struct {
	db *gorm.DB
}

func NewTeacherHandler(db *gorm.DB) *TeacherHandler {
	return &TeacherHandler{db: db}
}

func (h *TeacherHandler) GetTeachers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Получаем информацию о текущем пользователе
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	// Только админ может видеть список преподавателей
	if claims.Role != models.RoleAdmin {
		log.Printf(" User %s (role: %s) tried to access teachers without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	// Параметры пагинации
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 5
	}

	offset := (page - 1) * limit

	// Параметры сортировки
	sortBy := r.URL.Query().Get("sortBy")

	// Параметры фильтрации
	nameFilter := r.URL.Query().Get("name")
	surnameFilter := r.URL.Query().Get("surname")
	emailFilter := r.URL.Query().Get("email")

	// Создаем запрос с GORM
	query := h.db.Model(&models.Teacher{})

	// Применяем фильтрацию
	if nameFilter != "" {
		cleanName := strings.Trim(nameFilter, "*")
		query = query.Where("name ILIKE ?", "%"+cleanName+"%")
	}

	if surnameFilter != "" {
		cleanSurname := strings.Trim(surnameFilter, "*")
		query = query.Where("surname ILIKE ?", "%"+cleanSurname+"%")
	}

	if emailFilter != "" {
		cleanEmail := strings.Trim(emailFilter, "*")
		query = query.Where("email ILIKE ?", "%"+cleanEmail+"%")
	}

	// Получаем общее количество
	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		log.Printf(" Error counting teachers: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Применяем сортировки
	if sortBy != "" {
		if strings.HasPrefix(sortBy, "-") {
			field := strings.TrimPrefix(sortBy, "-")
			query = query.Order(field + " DESC")
		} else {
			query = query.Order(sortBy + " ASC")
		}
	} else {
		query = query.Order("id ASC")
	}

	// Применяем пагинацию
	var teachers []models.Teacher
	if err := query.Offset(offset).Limit(limit).Find(&teachers).Error; err != nil {
		log.Printf(" Error fetching teachers: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	totalPages := (int(totalItems) + limit - 1) / limit
	remainingCount := int(totalItems) - (page * limit)
	if remainingCount < 0 {
		remainingCount = 0
	}

	// Создаем отдельную структуру для ответа с преподавателями
	response := struct {
		Meta  models.Meta      `json:"meta"`
		Items []models.Teacher `json:"items"`
	}{
		Meta: models.Meta{
			TotalItems:     int(totalItems),
			TotalPages:     totalPages,
			CurrentPage:    page,
			PerPage:        limit,
			RemainingCount: remainingCount,
		},
		Items: teachers,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf(" Error encoding response: %v", err)
	}
}

func (h *TeacherHandler) CreateTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Проверяем права - только админ может создавать преподавателей
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf(" User %s (role: %s) tried to create teacher without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	log.Printf(" POST /api/teachers - Content-Type: %s, Content-Length: %d",
		r.Header.Get("Content-Type"), r.ContentLength)

	var createReq struct {
		Name    string `json:"name"`
		Surname string `json:"surname"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf(" Error reading request body: %v", err)
		http.Error(w, `{"error": "Cannot read request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📝 Request body: %s", string(body))

	if err := json.Unmarshal(body, &createReq); err != nil {
		log.Printf(" Error decoding JSON: %v", err)
		http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	log.Printf(" Creating teacher: Name='%s', Surname='%s', Email='%s', Phone='%s'",
		createReq.Name, createReq.Surname, createReq.Email, createReq.Phone)

	// Валидация
	if createReq.Name == "" || createReq.Surname == "" || createReq.Email == "" {
		log.Printf("Validation failed: Name, Surname and Email are required")
		http.Error(w, `{"error": "Name, surname and email are required"}`, http.StatusBadRequest)
		return
	}

	// Проверяем, существует ли преподаватель с таким email
	var existingTeacher models.Teacher
	if err := h.db.Where("email = ?", createReq.Email).First(&existingTeacher).Error; err == nil {
		log.Printf(" Teacher with email %s already exists", createReq.Email)
		http.Error(w, `{"error": "Teacher with this email already exists"}`, http.StatusConflict)
		return
	}

	// Создаем преподавателя
	teacher := models.Teacher{
		Name:    createReq.Name,
		Surname: createReq.Surname,
		Email:   createReq.Email,
		Phone:   createReq.Phone,
	}

	result := h.db.Create(&teacher)
	if result.Error != nil {
		log.Printf(" Database error creating teacher: %v", result.Error)
		http.Error(w, `{"error": "Failed to create teacher in database"}`, http.StatusInternalServerError)
		return
	}

	log.Printf(" Teacher created successfully with ID: %d", teacher.ID)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(teacher); err != nil {
		log.Printf(" Error encoding response: %v", err)
	}
}

func (h *TeacherHandler) UpdateTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Проверяем права - только админ может обновлять преподавателей
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf(" User %s (role: %s) tried to update teacher without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf(" Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid teacher ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("Updating teacher with ID: %d (by admin %s)", id, claims.Email)

	var updateReq struct {
		Name    string `json:"name"`
		Surname string `json:"surname"`
		Email   string `json:"email"`
		Phone   string `json:"phone"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		log.Printf(" Error decoding request body: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf(" Update data - Name: '%s', Surname: '%s', Email: '%s', Phone: '%s'",
		updateReq.Name, updateReq.Surname, updateReq.Email, updateReq.Phone)

	// Валидация
	if updateReq.Name == "" || updateReq.Surname == "" || updateReq.Email == "" {
		log.Printf(" Validation failed: Name, Surname and Email are required")
		http.Error(w, `{"error": "Name, surname and email are required"}`, http.StatusBadRequest)
		return
	}

	// Проверяем существование преподавателя
	var existingTeacher models.Teacher
	result := h.db.First(&existingTeacher, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf(" Teacher with ID %d not found", id)
			http.Error(w, `{"error": "Teacher not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("Error checking teacher existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Проверяем, не используется ли email другим преподавателем
	if updateReq.Email != existingTeacher.Email {
		var teacherWithSameEmail models.Teacher
		if err := h.db.Where("email = ? AND id != ?", updateReq.Email, id).First(&teacherWithSameEmail).Error; err == nil {
			log.Printf(" Email %s already used by another teacher", updateReq.Email)
			http.Error(w, `{"error": "Email already in use by another teacher"}`, http.StatusConflict)
			return
		}
	}

	// Обновляем преподавателя
	existingTeacher.Name = updateReq.Name
	existingTeacher.Surname = updateReq.Surname
	existingTeacher.Email = updateReq.Email
	existingTeacher.Phone = updateReq.Phone

	result = h.db.Save(&existingTeacher)
	if result.Error != nil {
		log.Printf(" Error updating teacher in database: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf(" Teacher updated successfully. Rows affected: %d", result.RowsAffected)

	// Получаем обновленного преподавателя
	var updatedTeacher models.Teacher
	h.db.First(&updatedTeacher, id)

	if err := json.NewEncoder(w).Encode(updatedTeacher); err != nil {
		log.Printf(" Error encoding response: %v", err)
	}
}

func (h *TeacherHandler) DeleteTeacher(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Проверяем права - только админ может удалять преподавателей
	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf(" User %s (role: %s) tried to delete teacher without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf(" Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid teacher ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🗑️ Deleting teacher with ID: %d (by admin %s)", id, claims.Email)

	// Проверяем существование преподавателя
	var teacher models.Teacher
	result := h.db.First(&teacher, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf(" Teacher with ID %d not found", id)
			http.Error(w, `{"error": "Teacher not found"}`, http.StatusNotFound)
			return
		}
		log.Printf(" Error checking teacher existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Удаляем преподавателя
	result = h.db.Delete(&teacher)
	if result.Error != nil {
		log.Printf(" Error deleting teacher: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf(" Teacher deleted successfully. Rows affected: %d", result.RowsAffected)
	w.WriteHeader(http.StatusNoContent)
}
