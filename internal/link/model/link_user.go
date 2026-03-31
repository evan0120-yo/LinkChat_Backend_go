package model

import "time"

type LinkUser struct {
	ID          string    `firestore:"id"`           // 對應 Auth User ID
	DisplayName string    `firestore:"display_name"` // 暱稱
	UpdatedAt   time.Time `firestore:"updated_at"`
	IsActive    bool      `firestore:"is_active"` // 帳號狀態
}
