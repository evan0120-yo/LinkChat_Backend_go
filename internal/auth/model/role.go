package model

// 定義 Role 為自訂字串型別，這樣可以防止隨便傳字串進來
type Role string

const (
	// 對應你說的：一般、VIP、Admin
	RoleUser  Role = "user"  // 一般用戶
	RoleVIP   Role = "vip"   // VIP
	RoleAdmin Role = "admin" // 管理員
)

// String 讓它可以直接轉回 string (雖然 Go 的 string type alias 本身就能轉，但實作 Stringer 介面是好習慣)
func (r Role) String() string {
	return string(r)
}

// Helper 方法：檢查是否為 Admin (寫法類似 Java Enum 裡的方法)
func (r Role) IsAdmin() bool {
	return r == RoleAdmin
}

// Helper 方法：檢查是否為 VIP 以上 (包含 Admin)
func (r Role) IsVIPOrAbove() bool {
	return r == RoleVIP || r == RoleAdmin
}

// Helper 方法：驗證存不存在這個 Role (用於 API 接收參數檢查)
func IsValidRole(r string) bool {
	switch Role(r) {
	case RoleUser, RoleVIP, RoleAdmin:
		return true
	default:
		return false
	}
}
