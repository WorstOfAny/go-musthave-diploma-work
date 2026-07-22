package models

import(
	"time"
)

type Order struct {
	UID *int `json:"-" db:"user_id"`
	Number string `json:"number"`
	Status string `json:"status"`
	UploadedAt time.Time `json:"uploaded_at"`
	Accrual *float64 `json:"accrual"`
}
