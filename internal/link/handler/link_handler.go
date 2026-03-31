package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	"github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
)

type LinkHandler struct {
	linkCommandUseCase cmdUseCase.LinkCommandUseCase
	linkQueryUseCase   qryUseCase.LinkQueryUseCase
}

func NewLinkHandler(
	linkCommandUseCase cmdUseCase.LinkCommandUseCase,
	linkQueryUseCase qryUseCase.LinkQueryUseCase,
) *LinkHandler {
	return &LinkHandler{
		linkCommandUseCase: linkCommandUseCase,
		linkQueryUseCase:   linkQueryUseCase,
	}
}

// RegisterRoutes 註冊路由與權限控制
func (h *LinkHandler) RegisterRoutes(router *gin.RouterGroup, mw *middleware.AuthMiddleware) {
	linkGroup := router.Group("/links")

	// 全面掛載 Token 驗證 Middleware
	linkGroup.Use(mw.VerifyToken())
	{
		// 搜尋用戶
		// POST /api/v1/links/search
		linkGroup.POST("/search", h.SearchUsers)

		// 申請好友
		// POST /api/v1/links/apply
		linkGroup.POST("/apply", h.ApplyLink)

		// 接受好友
		// POST /api/v1/links/accept
		linkGroup.POST("/accept", h.AcceptLink)

		// 拒絕好友
		// POST /api/v1/links/reject
		linkGroup.POST("/reject", h.RejectLink)

		// 解除好友 (Remove)
		// POST /api/v1/links/remove
		linkGroup.POST("/remove", h.RemoveLink)

		// [新增] 收回申請 (Cancel)
		// POST /api/v1/links/cancel
		linkGroup.POST("/cancel", h.CancelLink)

		// 查詢好友列表 (支援篩選)
		// GET /api/v1/links/list?filter=active
		linkGroup.GET("/list", h.GetLinkList)
	}
}

// ==========================================
// Methods
// ==========================================

// SearchUsers 搜尋用戶 (POST)
func (h *LinkHandler) SearchUsers(c *gin.Context) {
	var body struct {
		DisplayName string `json:"displayName" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	users, err := h.linkQueryUseCase.SearchUsers(c.Request.Context(), body.DisplayName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": users})
}

// ApplyLink 申請好友
func (h *LinkHandler) ApplyLink(c *gin.Context) {
	var body struct {
		TargetID string `json:"targetId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	requesterID := c.GetString(middleware.CtxKeyUserID)
	if requesterID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	applyReq := &req.ApplyLinkReq{
		RequesterID: requesterID,
		TargetID:    body.TargetID,
	}

	link, err := h.linkCommandUseCase.ApplyLink(c.Request.Context(), applyReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": link})
}

// AcceptLink 接受好友
func (h *LinkHandler) AcceptLink(c *gin.Context) {
	var body struct {
		LinkID string `json:"linkId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString(middleware.CtxKeyUserID)
	if operatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	acceptReq := &req.AcceptLinkReq{
		OperatorID: operatorID,
		LinkID:     body.LinkID,
	}

	if err := h.linkCommandUseCase.AcceptLink(c.Request.Context(), acceptReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// RejectLink 拒絕好友
func (h *LinkHandler) RejectLink(c *gin.Context) {
	var body struct {
		LinkID string `json:"linkId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString(middleware.CtxKeyUserID)
	if operatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	rejectReq := &req.RejectLinkReq{
		OperatorID: operatorID,
		LinkID:     body.LinkID,
	}

	if err := h.linkCommandUseCase.RejectLink(c.Request.Context(), rejectReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// RemoveLink 解除好友
func (h *LinkHandler) RemoveLink(c *gin.Context) {
	var body struct {
		LinkID string `json:"linkId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString(middleware.CtxKeyUserID)
	if operatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	removeReq := &req.RemoveLinkReq{
		OperatorID: operatorID,
		LinkID:     body.LinkID,
	}

	if err := h.linkCommandUseCase.RemoveLink(c.Request.Context(), removeReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// [新增] CancelLink 收回申請
func (h *LinkHandler) CancelLink(c *gin.Context) {
	var body struct {
		LinkID string `json:"linkId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	operatorID := c.GetString(middleware.CtxKeyUserID)
	if operatorID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	cancelReq := &req.CancelLinkReq{
		OperatorID: operatorID,
		LinkID:     body.LinkID,
	}

	if err := h.linkCommandUseCase.CancelLink(c.Request.Context(), cancelReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}

// GetLinkList 查詢好友列表
func (h *LinkHandler) GetLinkList(c *gin.Context) {
	userID := c.GetString(middleware.CtxKeyUserID)
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	filterParam := c.Query("filter")
	listReq := req.ListLinkReq{
		Filter: filterParam,
	}

	links, err := h.linkQueryUseCase.GetLinkList(c.Request.Context(), userID, listReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": links})
}
