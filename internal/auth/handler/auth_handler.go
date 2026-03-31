package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	// 引用 Middleware 和 Model (為了權限控制)
	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/object/req"
	cmdUC "github.com/evan0120-yo/linkchat-go/internal/auth/usecase/command"
	qryUC "github.com/evan0120-yo/linkchat-go/internal/auth/usecase/query"
)

// 1. Handler Struct
type AuthHandler struct {
	commandUseCase cmdUC.AuthCommandUseCase
	queryUseCase   qryUC.AuthQueryUseCase
}

// 2. Factory
func NewAuthHandler(
	cmd cmdUC.AuthCommandUseCase,
	qry qryUC.AuthQueryUseCase,
) *AuthHandler {
	return &AuthHandler{
		commandUseCase: cmd,
		queryUseCase:   qry,
	}
}

// 3. RegisterRoutes (修改簽章: 傳入 Middleware)
func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, mw *middleware.AuthMiddleware) {
	authGroup := router.Group("/auth")

	// ==============================
	// 公開路由 (Public)
	// ==============================
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)

	// ==============================
	// 保護路由 (Private)
	// ==============================
	// 建立子群組並掛載驗證器
	protected := authGroup.Group("/")
	protected.Use(mw.VerifyToken())
	{
		// 刪除用戶 (DELETE /citrus/auth/:id)
		// 這裡假設是管理員操作，所以加上 RequireRole(Admin)
		// 如果是刪除自己，則不需要 Admin 權限，但需要檢查 ID 是否匹配 Token
		protected.POST("/delete", h.DeleteUser)
	}
}

// ==========================================
// 4. Methods
// ==========================================

func (h *AuthHandler) Register(c *gin.Context) {
	var r req.RegisterReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	if err := h.commandUseCase.Register(ctx, &r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "user created successfully"})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var r req.LoginReq
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx := c.Request.Context()
	resp, err := h.queryUseCase.Login(ctx, &r)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) DeleteUser(c *gin.Context) {
	var req req.DeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 1. 從 Gin Context 取得當前登入者資訊 (由 Middleware 設定)
	actorID := c.GetString(middleware.CtxKeyUserID)

	// 注意：從 Context 拿出來的是 interface{}，要轉型成 model.Role
	roleVal, exists := c.Get(middleware.CtxKeyRole)
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	actorRole := roleVal.(model.Role)

	// 2. 呼叫 UseCase (傳入操作者資訊)
	err := h.commandUseCase.DeleteUser(c.Request.Context(), actorID, actorRole, req.UserID)
	if err != nil {
		// 這裡可以細分錯誤，例如如果是 permission denied 回傳 403
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "user deleted successfully"})
}
