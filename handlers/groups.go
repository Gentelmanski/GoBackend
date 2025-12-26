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

type GroupHandler struct {
	db *gorm.DB
}

func NewGroupHandler(db *gorm.DB) *GroupHandler {
	return &GroupHandler{db: db}
}

func (h *GroupHandler) GetGroups(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf("❌ User %s (role: %s) tried to access groups without permission",
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
	codeFilter := r.URL.Query().Get("code")

	query := h.db.Model(&models.Group{})

	if nameFilter != "" {
		cleanName := strings.Trim(nameFilter, "*")
		query = query.Where("name ILIKE ?", "%"+cleanName+"%")
	}

	if codeFilter != "" {
		cleanCode := strings.Trim(codeFilter, "*")
		query = query.Where("code ILIKE ?", "%"+cleanCode+"%")
	}

	var totalItems int64
	if err := query.Count(&totalItems).Error; err != nil {
		log.Printf("❌ Error counting groups: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

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

	var groups []models.Group
	if err := query.Offset(offset).Limit(limit).Find(&groups).Error; err != nil {
		log.Printf("❌ Error fetching groups: %v", err)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
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
		//Items: groups,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *GroupHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf("❌ User %s (role: %s) tried to create group without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	var createReq struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("❌ Error reading request body: %v", err)
		http.Error(w, `{"error": "Cannot read request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📝 Request body: %s", string(body))

	if err := json.Unmarshal(body, &createReq); err != nil {
		log.Printf("❌ Error decoding JSON: %v", err)
		http.Error(w, `{"error": "Invalid JSON format"}`, http.StatusBadRequest)
		return
	}

	log.Printf("➕ Creating group: Name='%s', Code='%s'", createReq.Name, createReq.Code)

	if createReq.Name == "" || createReq.Code == "" {
		log.Printf("❌ Validation failed: Name and Code are required")
		http.Error(w, `{"error": "Name and code are required"}`, http.StatusBadRequest)
		return
	}

	var existingGroup models.Group
	if err := h.db.Where("code = ?", createReq.Code).First(&existingGroup).Error; err == nil {
		log.Printf("❌ Group with code %s already exists", createReq.Code)
		http.Error(w, `{"error": "Group with this code already exists"}`, http.StatusConflict)
		return
	}

	group := models.Group{
		Name: createReq.Name,
		Code: createReq.Code,
	}

	result := h.db.Create(&group)
	if result.Error != nil {
		log.Printf("❌ Database error creating group: %v", result.Error)
		http.Error(w, `{"error": "Failed to create group in database"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Group created successfully with ID: %d", group.ID)

	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(group); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *GroupHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf("❌ User %s (role: %s) tried to update group without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("❌ Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid group ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🔄 Updating group with ID: %d (by admin %s)", id, claims.Email)

	var updateReq struct {
		Name string `json:"name"`
		Code string `json:"code"`
	}

	if err := json.NewDecoder(r.Body).Decode(&updateReq); err != nil {
		log.Printf("❌ Error decoding request body: %v", err)
		http.Error(w, `{"error": "Invalid request body"}`, http.StatusBadRequest)
		return
	}

	log.Printf("📝 Update data - Name: '%s', Code: '%s'", updateReq.Name, updateReq.Code)

	if updateReq.Name == "" || updateReq.Code == "" {
		log.Printf("❌ Validation failed: Name and Code are required")
		http.Error(w, `{"error": "Name and code are required"}`, http.StatusBadRequest)
		return
	}

	var existingGroup models.Group
	result := h.db.First(&existingGroup, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("❌ Group with ID %d not found", id)
			http.Error(w, `{"error": "Group not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("❌ Error checking group existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	if updateReq.Code != existingGroup.Code {
		var groupWithSameCode models.Group
		if err := h.db.Where("code = ? AND id != ?", updateReq.Code, id).First(&groupWithSameCode).Error; err == nil {
			log.Printf("❌ Code %s already used by another group", updateReq.Code)
			http.Error(w, `{"error": "Code already in use by another group"}`, http.StatusConflict)
			return
		}
	}

	existingGroup.Name = updateReq.Name
	existingGroup.Code = updateReq.Code

	result = h.db.Save(&existingGroup)
	if result.Error != nil {
		log.Printf("❌ Error updating group in database: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Group updated successfully. Rows affected: %d", result.RowsAffected)

	var updatedGroup models.Group
	h.db.First(&updatedGroup, id)

	if err := json.NewEncoder(w).Encode(updatedGroup); err != nil {
		log.Printf("❌ Error encoding response: %v", err)
	}
}

func (h *GroupHandler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	claims := middleware.GetUserClaims(r.Context())
	if claims == nil {
		http.Error(w, `{"error": "Not authenticated"}`, http.StatusUnauthorized)
		return
	}

	if claims.Role != models.RoleAdmin {
		log.Printf("❌ User %s (role: %s) tried to delete group without permission",
			claims.Email, claims.Role)
		http.Error(w, `{"error": "Insufficient permissions"}`, http.StatusForbidden)
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		log.Printf("❌ Error converting id to int: %v", err)
		http.Error(w, `{"error": "Invalid group ID"}`, http.StatusBadRequest)
		return
	}

	log.Printf("🗑️ Deleting group with ID: %d (by admin %s)", id, claims.Email)

	var group models.Group
	result := h.db.First(&group, id)
	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			log.Printf("❌ Group with ID %d not found", id)
			http.Error(w, `{"error": "Group not found"}`, http.StatusNotFound)
			return
		}
		log.Printf("❌ Error checking group existence: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	result = h.db.Delete(&group)
	if result.Error != nil {
		log.Printf("❌ Error deleting group: %v", result.Error)
		http.Error(w, `{"error": "Internal server error"}`, http.StatusInternalServerError)
		return
	}

	log.Printf("✅ Group deleted successfully. Rows affected: %d", result.RowsAffected)
	w.WriteHeader(http.StatusNoContent)
}
