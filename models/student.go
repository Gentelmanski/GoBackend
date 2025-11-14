package models

type Student struct {
	ID      *uint  `json:"id,omitempty" gorm:"primaryKey;autoIncrement"`
	Name    string `json:"name" gorm:"size:100;not null"`
	Surname string `json:"surname" gorm:"size:100;not null"`
}

// TableName указывает имя таблицы в БД
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
	Direction string `json:"direction"` // "asc" или "desc"
}
