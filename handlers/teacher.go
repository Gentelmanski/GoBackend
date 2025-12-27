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

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf("❌ User %s (role: %s) tried to access teachers without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 {
		limit = 5
	}

	offset := (page - 1) * limit

	sortBy := r.URL.Query().Get("sortBy")
	nameFilter := r.URL.Query().Get("name")
	surnameFilter := r.URL.Query().Get("surname")
	emailFilter := r.URL.Query().Get("email")

	// Создаем базовый запрос
	query := h.db.Model(&models.Teacher{})

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

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		log.Printf("❌ Error counting teachers: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Сортируем и применяем пагинацию
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

	var teachers []models.Teacher
	if err := query.Offset(offset).Limit(limit).Find(&teachers).Error; err != nil {
		log.Printf("❌ Error fetching teachers: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Загружаем группы для каждого преподавателя отдельно
	for i := range teachers {
		if err := h.db.Model(&teachers[i]).Association("Groups").Find(&teachers[i].Groups); err != nil {
			log.Printf("❌ Error loading groups for teacher %d: %v", teachers[i].ID, err)
		}
	}

	totalPages := (int(totalItems) + limit - 1) / limit
	remainingCount := int(totalItems) - (page * limit)
	if remainingCount < 0 {
		remainingCount = 0
	}

	response := models.PaginatedResponse{
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
		log.Printf("❌ Error encoding response: %v", err)
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

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("❌ Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid teacher ID"}`, http.StatusBadRequest)
		return
	}

	var teacher models.Teacher
	result := h.db.Preload("Groups").First(&teacher, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			http.Error(w, `{"error": "Teacher not found"}`, http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	var updateReq struct {
		Name    string         `json:"name"`
		Surname string         `json:"surname"`
		Email   string         `json:"email"`
		Phone   string         `json:"phone"`
		Groups  []models.Group `json:"groups"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		log.Printf("❌ Error decoding request body: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	// Обновляем основные поля
	teacher.Name = updateReq.Name
	teacher.Surname = updateReq.Surname
	teacher.Email = updateReq.Email
	teacher.Phone = updateReq.Phone

	// Обновляем связи с группами
	if updateReq.Groups != nil {
		// Получаем ID групп из запроса
		var groupIDs []uint
		for _, group := range updateReq.Groups {
			if group.ID > 0 {
				groupIDs = append(groupIDs, group.ID)
			}
		}

		// Находим группы по ID
		var groups []models.Group
		if len(groupIDs) > 0 {
			if err := h.db.Where("id IN ?", groupIDs).Find(&groups).Error; err != nil {
				log.Printf("❌ Error finding groups: %v", err)
				http.Error(w, `{"error": "Invalid group IDs"}`, http.StatusBadRequest)
				return
			}
		}

		// Обновляем связи
		if err := h.db.Model(&teacher).Association("Groups").Replace(&groups); err != nil {
			log.Printf("❌ Error updating teacher groups: %v", err)
			http.Error(w, `{"error": "Failed to update groups"}`, http.StatusInternalServerError)
			return
		}
	}

	// Сохраняем изменения
	if err := h.db.Save(&teacher).Error; err != nil {
		log.Printf("❌ Error updating teacher: %v", err)
		http.Error(w, `{"error": "Failed to update teacher"}`, http.StatusInternalServerError)
		return
	}

	// Подгружаем группы для ответа
	h.db.Preload("Groups").First(&teacher, teacher.ID)

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(teacher); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
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
