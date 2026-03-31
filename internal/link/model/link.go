package model

import "time"

type LinkStatus string

const (
	StatusPending LinkStatus = "pending" // 申請中
	StatusActive  LinkStatus = "active"  // 已連結 (好友)
	StatusBlocked LinkStatus = "blocked" // 封鎖
)

type Link struct {
	ID string `firestore:"id"`

	// 關係雙方
	RequesterID string `firestore:"requester_id"` // 主動方
	TargetID    string `firestore:"target_id"`    // 被動方

	// 關鍵欄位: 查詢用
	// 內容: [RequesterID, TargetID]
	// 用途: Where("participants", "array-contains", "Me") -> 一次撈出所有相關連結
	Participants []string `firestore:"participants"`

	// 當前狀態
	Status LinkStatus `firestore:"status"`

	CreatedAt time.Time `firestore:"created_at"`
	UpdatedAt time.Time `firestore:"updated_at"`
}
