package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
	profileUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/query"
)

type ProfileHandler struct {
	profileCommandUseCase cmdUseCase.ProfileCommandUseCase
	profileQueryUseCase   qryUseCase.ProfileQueryUseCase
}

func NewProfileHandler(
	profileCommandUseCase cmdUseCase.ProfileCommandUseCase,
	profileQueryUseCase qryUseCase.ProfileQueryUseCase,
) *ProfileHandler {
	return &ProfileHandler{
		profileCommandUseCase: profileCommandUseCase,
		profileQueryUseCase:   profileQueryUseCase,
	}
}

func (h *ProfileHandler) RegisterRoutes(router *gin.RouterGroup, mw *middleware.AuthMiddleware) {
	profileGroup := router.Group("/profiles")
	profileGroup.Use(mw.VerifyToken())
	{
		profileGroup.PUT("/notes", h.SaveSubjectNotes)
		profileGroup.PUT("/tags", h.SaveSubjectTags)
		profileGroup.GET("/context", h.GetSubjectProfileContext)
		profileGroup.GET("/tag-catalog", h.GetTagCatalog)
	}
}

func (h *ProfileHandler) SaveSubjectNotes(c *gin.Context) {
	var request reqObj.SaveSubjectNotesReq
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID := c.GetString(middleware.CtxKeyUserID)
	if ownerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	data, err := h.profileCommandUseCase.SaveSubjectNotes(c.Request.Context(), ownerID, &request)
	if err != nil {
		writeProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *ProfileHandler) SaveSubjectTags(c *gin.Context) {
	var request reqObj.SaveSubjectTagsReq
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ownerID := c.GetString(middleware.CtxKeyUserID)
	if ownerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	data, err := h.profileCommandUseCase.SaveSubjectTags(c.Request.Context(), ownerID, &request)
	if err != nil {
		writeProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *ProfileHandler) GetSubjectProfileContext(c *gin.Context) {
	ownerID := c.GetString(middleware.CtxKeyUserID)
	if ownerID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	subjectID := c.Query("subjectId")
	if strings.TrimSpace(subjectID) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "subjectId is required"})
		return
	}
	data, err := h.profileQueryUseCase.GetSubjectProfileContext(c.Request.Context(), ownerID, subjectID)
	if err != nil {
		writeProfileError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func (h *ProfileHandler) GetTagCatalog(c *gin.Context) {
	data, err := h.profileQueryUseCase.GetTagCatalog(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": data})
}

func writeProfileError(c *gin.Context, err error) {
	c.JSON(profileErrorStatus(err), gin.H{"error": err.Error()})
}

func profileErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if errors.Is(err, profileUseCase.ErrSubjectNotAccessible) {
		return http.StatusForbidden
	}
	if isProfileValidationError(err) {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}

func isProfileValidationError(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case errors.Is(err, profileUseCase.ErrOwnerIDRequired),
		errors.Is(err, profileUseCase.ErrSubjectIDRequired),
		errors.Is(err, profileUseCase.ErrSubjectIsCurrentUser):
		return true
	}

	message := strings.TrimSpace(err.Error())
	switch {
	case strings.HasPrefix(message, "line ") && strings.Contains(message, " exceeds "):
		return true
	case strings.HasPrefix(message, "note lines cannot exceed "):
		return true
	case message == "groupKey and tagKey are required":
		return true
	case strings.HasPrefix(message, "tag group not found:"):
		return true
	case strings.HasPrefix(message, "tag not found or inactive:"):
		return true
	case strings.HasPrefix(message, "tag group only allows single selection:"):
		return true
	default:
		return false
	}
}
