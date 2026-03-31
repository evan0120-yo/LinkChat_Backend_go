package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	// 引用 model 是為了使用 Role Enum
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
)

// 定義 Context Key 常數，避免魔法字串 (Magic String)
const (
	CtxKeyUserID = "userID"
	CtxKeyRole   = "userRole"
)

// AuthMiddleware
type AuthMiddleware struct {
	jwtSecret []byte
}

// Factory
func NewAuthMiddleware(secret string) *AuthMiddleware {
	return &AuthMiddleware{
		jwtSecret: []byte(secret),
	}
}

// ==========================================
// 1. 驗證層 (Authentication): 只確認 Token 有效
// ==========================================
func (m *AuthMiddleware) VerifyToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 從 Header 拿 Token
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		// 格式通常是 "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization format"})
			return
		}
		tokenString := parts[1]

		// 2. 解析 Token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// 驗證簽名演算法
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return m.jwtSecret, nil
		})

		// 3. 處理解析結果
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// 4. 取出 Claims 並塞入 Context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			// 注意: JWT 解析出來的數字通常是 float64
			if sub, ok := claims["sub"].(string); ok {
				c.Set(CtxKeyUserID, sub)
			}
			if roleStr, ok := claims["role"].(string); ok {
				// 轉成我們定義的 Role 型別
				c.Set(CtxKeyRole, model.Role(roleStr))
			}
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}

		// 5. 放行
		c.Next()
	}
}

// ==========================================
// 2. 授權層 (Authorization): 檢查 Role
// ==========================================
// 這是一個 "Higher-Order Function"，你傳入需要的權限，它回傳一個 Handler
func (m *AuthMiddleware) RequireRole(requiredRole model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 從 Context 拿出剛剛 VerifyToken 塞進去的 Role
		roleVal, exists := c.Get(CtxKeyRole)
		if !exists {
			// 如果沒有 Role，代表可能沒經過 VerifyToken，或是 Token 裡沒 Role
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		userRole := roleVal.(model.Role)

		// 2. 權限判斷邏輯
		// 這裡示範簡單的 "必須相等"，或是 "Admin 無敵"
		if userRole == model.RoleAdmin {
			c.Next() // Admin 通行無阻
			return
		}

		if userRole != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden: insufficient permissions"})
			return
		}

		c.Next()
	}
}
