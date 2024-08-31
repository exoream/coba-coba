package model

import (
	"time"
)

type Transaction struct {
	ID        int    `json:"id"`
	UserID    int       `json:"user_id"`
	AdminID   int       `json:"admin_id"`
	ExpiresAt time.Time `json:"expires_at"`
}
