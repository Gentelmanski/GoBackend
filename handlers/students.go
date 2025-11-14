package handlers

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"student-backend/models"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

type StudentHandler struct {
	db *gorm.DB
}

func NewStudentHandler(db *gorm.DB) *StudentHandler {
	return &StudentHandler{db: db}
}

func (h *StudentHandler) GetStudents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

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

	// Создаем запрос с GORM
	query := h.db.Model(&models.Student{})

	// Применяем фильтрацию
	if nameFilter != "" {
		cleanName := strings.Trim(nameFilter, "*")
		query = query.Where("name ILIKE ?", "%"+cleanName+"%")
	}

	if surnameFilter != "" {
		cleanSurname := strings.Trim(surnameFilter, "*")
		query = query.Where("surname ILIKE ?", "%"+cleanSurname+"%")
	}

	// Получаем общее количество
	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		log.Printf("❌ Error counting students: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Применяем сортировку
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
	var students []models.Student
	if err := query.Offset(offset).Limit(limit).Find(&students).Error; err != nil {
		log.Printf("❌ Error fetching students: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Рассчитываем метаданные
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
		Items: students,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *StudentHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	log.Printf("📨 POST /api/students - Content-Type: %s, Content-Length: %d",
		r.Header.Get("Content-Type"), r.ContentLength)

	var student models.Student
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading request body: %v", err)
		http.Error(w, `{"error": "Cannot read request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📝 Request body: %s", string(body))

	if err := json.Unmarshal(body, &student); err != nil {
		log.Printf("❌ Error decoding JSON: %v", err)
		http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	log.Printf("➕ Creating student: Name='%s', Surname='%s'", student.Name, student.Surname)

	// Валидация
	if student.Name == "" || student.Surname == "" {
		log.Printf("❌ Validation failed: Name or Surname is empty")
		http.Error(w, `{"error": "Name and surname are required"}`, http.StatusBadRequest)
		return
	}

	// Создаем студента с GORM
	result := h.db.Create(&student)
	if result.Error != nil {
		log.Printf("❌ Database error creating student: %v", result.Error)
		http.Error(w, `{"error": "Failed to create student in database"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Student created successfully with ID: %d", student.ID)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(student); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *StudentHandler) UpdateStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("❌ Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid student ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Updating student with ID: %d", id)

	var student models.Student
	if err := json.NewDecoder(r.Body).Decode(&student); err != nil {
		log.Printf("❌ Error decoding request body: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📝 Update data - Name: '%s', Surname: '%s'", student.Name, student.Surname)

	// Валидация
	if student.Name == "" || student.Surname == "" {
		log.Printf("❌ Validation failed: Name or Surname is empty")
		http.Error(w, `{"error": "Name and surname are required"}`, http.StatusBadRequest)
		return
	}

	// Проверяем существование студента
	var existingStudent models.Student
	result := h.db.First(&existingStudent, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("❌ Student with ID %d not found", id)
			http.Error(w, `{"error": "Student not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("❌ Error checking student existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Обновляем студента с GORM
	updateData := models.Student{
		Name:    student.Name,
		Surname: student.Surname,
	}

	result = h.db.Model(&existingStudent).Updates(updateData)
	if result.Error != nil {
		log.Printf("❌ Error updating student in database: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Student updated successfully. Rows affected: %d", result.RowsAffected)

	// Получаем обновленного студента
	var updatedStudent models.Student
	h.db.First(&updatedStudent, id)

	if err := json.NewEncoder(w).Encode(updatedStudent); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *StudentHandler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("❌ Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid student ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🗑️ Deleting student with ID: %d", id)

	// Проверяем существование студента
	var student models.Student
	result := h.db.First(&student, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("❌ Student with ID %d not found", id)
			http.Error(w, `{"error": "Student not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("❌ Error checking student existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Удаляем студента с GORM
	result = h.db.Delete(&student)
	if result.Error != nil {
		log.Printf("❌ Error deleting student: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Student deleted successfully. Rows affected: %d", result.RowsAffected)
	w.WriteHeader(http.StatusNoContent)
}
