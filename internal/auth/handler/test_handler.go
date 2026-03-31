package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
)

// TestHandler 專門用來測試驗證機制的 Handler
type TestHandler struct {
}

// Factory
func NewTestHandler() *TestHandler {
	return &TestHandler{}
}

func (h *TestHandler) RegisterRoutes(router *gin.RouterGroup, mw *middleware.AuthMiddleware) {
	// 1. 建立基礎群組 /citrus/test (此時還沒掛 Middleware)
	apiGroup := router.Group("/test")

	// ==========================================
	// 公開路由 (Public) - 不需 Token
	// ==========================================
	// POST /citrus/test/ping
	apiGroup.POST("/ping", h.Ping)

	// ==========================================
	// 私有路由 (Private) - 需 Token
	// ==========================================
	// 我們建立一個子群組 (路徑不變，還是繼承 /test)，但在這裡掛上 Middleware
	privateGroup := apiGroup.Group("/")
	privateGroup.Use(mw.VerifyToken())
	{
		// 1. 一般測試 (只要有登入)
		// POST /citrus/test/profile
		privateGroup.POST("/profile", h.GetProfile)

		// 2. 進階權限測試 (只有 Admin)
		// POST /citrus/test/system
		privateGroup.POST("/system", mw.RequireRole(model.RoleAdmin), h.DeleteSystem)
	}
}

// Ping (公開測試)
func (h *TestHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Pong! 這是公開的 POST 測試，不用登入也能看",
	})
}

// GetProfile 測試獲取 User 資訊
func (h *TestHandler) GetProfile(c *gin.Context) {
	// 從 Context 取出 Middleware 解析的資料
	userID, _ := c.Get(middleware.CtxKeyUserID)
	role, _ := c.Get(middleware.CtxKeyRole)

	c.JSON(http.StatusOK, gin.H{
		"message": "驗證成功！你是登入狀態",
		"user_id": userID,
		"role":    role,
	})
}

// DeleteSystem 測試 Admin 權限
func (h *TestHandler) DeleteSystem(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Admin 操作成功，系統已刪除 (假)",
	})
}
