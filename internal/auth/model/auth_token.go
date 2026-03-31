package model

// AuthToken 定義 Token 回傳格式
// 它是 Auth 領域的核心 Value Object，所以放在 model 是對的
type AuthToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}
