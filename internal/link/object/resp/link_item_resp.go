package resp

type LinkItemResp struct {
	LinkID      string `json:"linkId"`      // 操作用 (Accept/Delete)
	UserID      string `json:"userId"`      // 對方 ID
	DisplayName string `json:"displayName"` // 對方暱稱
	Status      string `json:"status"`      // "active", "pending_received", "pending_sent"
	Direction   string `json:"direction"`   // "incoming", "outgoing", "none" (已是好友)
}
