package models

type Student struct {
	ID      *int   `json:"id,omitempty" db:"id"`
	Name    string `json:"name" db:"name"`
	Surname string `json:"surname" db:"surname"`
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
