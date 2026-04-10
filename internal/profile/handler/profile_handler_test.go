package handler

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
	respObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/resp"
	profileUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/query"
)

func TestSaveSubjectNotesReturnsInternalServerErrorForUnexpectedUseCaseFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProfileHandler(
		&fakeProfileCommandUseCase{saveSubjectNotesErr: errors.New("firestore timeout")},
		&fakeProfileQueryUseCase{},
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/profiles/notes", strings.NewReader(`{"subjectId":"subject-1","lines":["note"]}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set(middleware.CtxKeyUserID, "owner-1")

	handler.SaveSubjectNotes(ctx)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetSubjectProfileContextReturnsForbiddenWhenSubjectNotAccessible(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProfileHandler(
		&fakeProfileCommandUseCase{},
		&fakeProfileQueryUseCase{getContextErr: profileUseCase.ErrSubjectNotAccessible},
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/profiles/context?subjectId=subject-1", nil)
	ctx.Set(middleware.CtxKeyUserID, "owner-1")

	handler.GetSubjectProfileContext(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestGetSubjectProfileContextRejectsBlankSubjectID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewProfileHandler(
		&fakeProfileCommandUseCase{},
		&fakeProfileQueryUseCase{},
	)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/profiles/context", nil)
	ctx.Set(middleware.CtxKeyUserID, "owner-1")

	handler.GetSubjectProfileContext(ctx)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d body=%s", recorder.Code, recorder.Body.String())
	}
}

type fakeProfileCommandUseCase struct {
	saveSubjectNotesErr error
	saveSubjectTagsErr  error
}

func (f *fakeProfileCommandUseCase) SaveSubjectNotes(_ context.Context, _ string, _ *reqObj.SaveSubjectNotesReq) (*respObj.SubjectProfileResp, error) {
	return nil, f.saveSubjectNotesErr
}

func (f *fakeProfileCommandUseCase) SaveSubjectTags(_ context.Context, _ string, _ *reqObj.SaveSubjectTagsReq) (*respObj.SubjectProfileResp, error) {
	return nil, f.saveSubjectTagsErr
}

type fakeProfileQueryUseCase struct {
	getContextErr error
}

func (f *fakeProfileQueryUseCase) GetSubjectProfileContext(_ context.Context, _ string, _ string) (*respObj.SubjectProfileResp, error) {
	return nil, f.getContextErr
}

func (f *fakeProfileQueryUseCase) GetTagCatalog(_ context.Context) (*respObj.TagCatalogResp, error) {
	return nil, nil
}

var (
	_ cmdUseCase.ProfileCommandUseCase = (*fakeProfileCommandUseCase)(nil)
	_ qryUseCase.ProfileQueryUseCase   = (*fakeProfileQueryUseCase)(nil)
)
