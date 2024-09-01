package model

import (
	"time"
)

type Transaction struct {
	ID        int    `json:"id"`
	UserID    int       `json:"user_id"`
	AdminID   int       `json:"admin_id"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	ExpiresAt time.Time `json:"expires_at"`
}
