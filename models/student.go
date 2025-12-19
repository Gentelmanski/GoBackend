package models

import (
	"time"

	"gorm.io/gorm"
)

type Student struct {
	ID        uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name      string         `json:"name" gorm:"size:100;not null"`
	Surname   string         `json:"surname" gorm:"size:100;not null"`
	Email     string         `json:"email,omitempty" gorm:"size:255"`
	GroupID   *uint          `json:"group_id,omitempty"`
	Group     *Group         `json:"group,omitempty" gorm:"foreignKey:GroupID"`
	UserID    *uint          `json:"user_id,omitempty" gorm:"unique"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Student) TableName() string {
	return "students"
}

type PaginatedResponse struct {
	Meta  Meta      `json:"meta"`
	Items []Student `json:"items"`
}

type Meta struct {
	TotalItems     int `json:"total_items"`
	TotalPages     int `json:"total_pages"`
	CurrentPage    int `json:"current_page"`
	PerPage        int `json:"per_page"`
	RemainingCount int `json:"remaining_count"`
}

type SortConfig struct {
	Active    string `json:"active"`
	Direction string `json:"direction"`
}
