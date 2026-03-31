package model

import "time"

type User struct {
	ID          string    `firestore:"id"` // 系統唯一識別碼 (Primary Key)
	Email       string    `firestore:"email"`
	Password    string    `firestore:"password" json:"-"`
	DisplayName string    `firestore:"display_name"`
	Role        Role      `firestore:"role"`       // 角色
	CreatedAt   time.Time `firestore:"created_at"` // 建立時間
	IsActive    bool      `firestore:"is_active"`  // 帳號狀態
}
