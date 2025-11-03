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
	"github.com/jmoiron/sqlx"
)

type StudentHandler struct {
	db *sqlx.DB
}

func NewStudentHandler(db *sqlx.DB) *StudentHandler {
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
	var orderBy string
	if sortBy != "" {
		if strings.HasPrefix(sortBy, "-") {
			field := strings.TrimPrefix(sortBy, "-")
			orderBy = field + " DESC"
		} else {
			orderBy = sortBy + " ASC"
		}
	} else {
		orderBy = "id ASC"
	}

	// Параметры фильтрации
	nameFilter := r.URL.Query().Get("name")
	surnameFilter := r.URL.Query().Get("surname")

	// Базовый запрос
	baseQuery := "FROM students WHERE 1=1"
	countQuery := "SELECT COUNT(*) " + baseQuery
	dataQuery := "SELECT id, name, surname " + baseQuery

	var args []interface{}
	argCount := 0

	// Добавляем условия фильтрации
	if nameFilter != "" {
		argCount++
		baseQuery += " AND name ILIKE $" + strconv.Itoa(argCount)
		args = append(args, "%"+strings.Trim(nameFilter, "*")+"%")
	}

	if surnameFilter != "" {
		argCount++
		baseQuery += " AND surname ILIKE $" + strconv.Itoa(argCount)
		args = append(args, "%"+strings.Trim(surnameFilter, "*")+"%")
	}

	// Обновляем запросы с учетом фильтров
	countQuery = "SELECT COUNT(*) " + baseQuery
	dataQuery = "SELECT id, name, surname " + baseQuery + " ORDER BY " + orderBy +
		" LIMIT $" + strconv.Itoa(argCount+1) + " OFFSET $" + strconv.Itoa(argCount+2)

	// Получаем общее количество
	var totalItems int
	err := h.db.Get(&totalItems, countQuery, args...)
	if err != nil {
		log.Printf("Error counting students: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Получаем данные
	args = append(args, limit, offset)
	var students []models.Student
	err = h.db.Select(&students, dataQuery, args...)
	if err != nil {
		log.Printf("Error fetching students: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	// Рассчитываем метаданные
	totalPages := (totalItems + limit - 1) / limit
	remainingCount := totalItems - (page * limit)
	if remainingCount < 0 {
		remainingCount = 0
	}

	response := models.PaginatedResponse{
		Meta: models.Meta{
			TotalItems:     totalItems,
			TotalPages:     totalPages,
			CurrentPage:    page,
			PerPage:        limit,
			RemainingCount: remainingCount,
		},
		Items: students,
	}

	json.NewEncoder(w).Encode(response)
}

func (h *StudentHandler) CreateStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Логируем весь запрос для отладки
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

	// Декодируем JSON
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

	// Проверяем подключение к базе данных
	if err := h.db.Ping(); err != nil {
		log.Printf("❌ Database connection error: %v", err)
		http.Error(w, `{"error": "Database connection failed"}`, http.StatusInternalServerError)
		return
	}

	query := `INSERT INTO students (name, surname) VALUES ($1, $2) RETURNING id`
	var id int
	err = h.db.QueryRow(query, student.Name, student.Surname).Scan(&id)
	if err != nil {
		log.Printf("❌ Database error creating student: %v", err)
		log.Printf("❌ Query: %s, Params: %s, %s", query, student.Name, student.Surname)
		http.Error(w, `{"error": "Failed to create student in database"}`, http.StatusInternalServerError)
		return
	}

	student.ID = &id
	log.Printf("✅ Student created successfully with ID: %d", id)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(student); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

// Обновление студента
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
	var exists bool
	err = h.db.QueryRow("SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		log.Printf("❌ Error checking student existence: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	if !exists {
		log.Printf("❌ Student with ID %d not found", id)
		http.Error(w, `{"error": "Student not found"}`, http.StatusNotFound)
		return
	}

	// Обновляем студента (ТОЛЬКО name и surname)
	query := `UPDATE students SET name = $1, surname = $2 WHERE id = $3`
	result, err := h.db.Exec(query, student.Name, student.Surname, id)
	if err != nil {
		log.Printf("❌ Error updating student in database: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("⚠️ Error getting rows affected: %v", err)
	} else {
		log.Printf("✅ Student updated successfully. Rows affected: %d", rowsAffected)
	}

	// Возвращаем обновленного студента
	student.ID = &id
	if err := json.NewEncoder(w).Encode(student); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *StudentHandler) DeleteStudent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		http.Error(w, `{"error": "Invalid student ID"}`, http.StatusBadRequest)
		return
	}

	// Проверяем существование студента
	var exists bool
	err = h.db.Get(&exists, "SELECT EXISTS(SELECT 1 FROM students WHERE id = $1)", id)
	if err != nil {
		log.Printf("Error checking student existence: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	if !exists {
		http.Error(w, `{"error": "Student not found"}`, http.StatusNotFound)
		return
	}

	_, err = h.db.Exec("DELETE FROM students WHERE id = $1", id)
	if err != nil {
		log.Printf("Error deleting student: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
