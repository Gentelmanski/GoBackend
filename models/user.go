package models

import (
	"time"

	"gorm.io/gorm"
)

// Роли пользователей
const (
	RoleAdmin   = "admin"
	RoleTeacher = "teacher"
	RoleStudent = "student"
)

type User struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Email     string         `json:"email" gorm:"unique;not null;size:255"`
	Password  string         `json:"-" gorm:"not null;size:255"`
	Role      string         `json:"role" gorm:"not null;size:50"`
	StudentID *uint          `json:"student_id,omitempty" gorm:"unique"`
	TeacherID *uint          `json:"teacher_id,omitempty" gorm:"unique"`
	Student   *Student       `json:"student,omitempty" gorm:"foreignKey:StudentID"`
	Teacher   *Teacher       `json:"teacher,omitempty" gorm:"foreignKey:TeacherID"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (User) TableName() string {
	return "users"
}

// Запросы для аутентификации
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=admin teacher student"`
}
