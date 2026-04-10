package command

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	linkQryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
	respObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/resp"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/profile/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/profile/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/service/validator"
	profileUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase"
)

type ProfileCommandUseCase interface {
	SaveSubjectNotes(ctx context.Context, ownerID string, request *reqObj.SaveSubjectNotesReq) (*respObj.SubjectProfileResp, error)
	SaveSubjectTags(ctx context.Context, ownerID string, request *reqObj.SaveSubjectTagsReq) (*respObj.SubjectProfileResp, error)
}

type profileCommandUseCase struct {
	client                   *firestore.Client
	validator                validator.ProfileValidator
	subjectProfileCommandSvc cmdService.SubjectProfileCommandService
	subjectProfileQuerySvc   qryService.SubjectProfileQueryService
	tagCatalogQuerySvc       qryService.TagCatalogQueryService
	linkQueryUseCase         linkQryUseCase.LinkQueryUseCase
	nowFn                    func() time.Time
}

func NewProfileCommandUseCase(
	client *firestore.Client,
	validator validator.ProfileValidator,
	subjectProfileCommandSvc cmdService.SubjectProfileCommandService,
	subjectProfileQuerySvc qryService.SubjectProfileQueryService,
	tagCatalogQuerySvc qryService.TagCatalogQueryService,
	linkQueryUseCase linkQryUseCase.LinkQueryUseCase,
) ProfileCommandUseCase {
	return &profileCommandUseCase{
		client:                   client,
		validator:                validator,
		subjectProfileCommandSvc: subjectProfileCommandSvc,
		subjectProfileQuerySvc:   subjectProfileQuerySvc,
		tagCatalogQuerySvc:       tagCatalogQuerySvc,
		linkQueryUseCase:         linkQueryUseCase,
		nowFn:                    time.Now,
	}
}

func (u *profileCommandUseCase) SaveSubjectNotes(ctx context.Context, ownerID string, request *reqObj.SaveSubjectNotesReq) (*respObj.SubjectProfileResp, error) {
	if err := profileUseCase.EnsureAccessibleSubject(ctx, u.linkQueryUseCase, ownerID, request.SubjectID); err != nil {
		return nil, err
	}

	noteLines, err := u.validator.NormalizeNoteLines(request.Lines)
	if err != nil {
		return nil, err
	}

	profileID := profileUseCase.BuildSubjectProfileID(ownerID, request.SubjectID)
	var savedProfile *model.SubjectProfile
	if err := u.runSubjectProfileTx(ctx, func(ctx context.Context, querySvc qryService.SubjectProfileQueryService, commandSvc cmdService.SubjectProfileCommandService) error {
		existingProfile, err := querySvc.GetSubjectProfileByID(ctx, profileID)
		if err != nil {
			return err
		}

		selectedTags := []model.SelectedTag{}
		if existingProfile != nil && len(existingProfile.SelectedTags) > 0 {
			selectedTags = append(selectedTags, existingProfile.SelectedTags...)
		}

		if profileUseCase.IsEmptyProfile(noteLines, selectedTags) {
			savedProfile = nil
			return commandSvc.DeleteSubjectProfile(ctx, profileID)
		}

		profile := profileUseCase.BuildSubjectProfileModel(ownerID, request.SubjectID, noteLines, selectedTags, u.nowFn())
		if err := commandSvc.SaveSubjectProfile(ctx, profile); err != nil {
			return err
		}
		savedProfile = profile
		return nil
	}); err != nil {
		return nil, err
	}

	return profileUseCase.ToSubjectProfileResponse(savedProfile, request.SubjectID), nil
}

func (u *profileCommandUseCase) SaveSubjectTags(ctx context.Context, ownerID string, request *reqObj.SaveSubjectTagsReq) (*respObj.SubjectProfileResp, error) {
	if err := profileUseCase.EnsureAccessibleSubject(ctx, u.linkQueryUseCase, ownerID, request.SubjectID); err != nil {
		return nil, err
	}

	catalog, err := u.tagCatalogQuerySvc.GetActiveTagCatalog(ctx)
	if err != nil {
		return nil, err
	}

	selectedTags, err := u.validator.ValidateAndNormalizeSelectedTags(request.Selected, catalog)
	if err != nil {
		return nil, err
	}

	profileID := profileUseCase.BuildSubjectProfileID(ownerID, request.SubjectID)
	var savedProfile *model.SubjectProfile
	if err := u.runSubjectProfileTx(ctx, func(ctx context.Context, querySvc qryService.SubjectProfileQueryService, commandSvc cmdService.SubjectProfileCommandService) error {
		existingProfile, err := querySvc.GetSubjectProfileByID(ctx, profileID)
		if err != nil {
			return err
		}

		noteLines := []string{}
		if existingProfile != nil && len(existingProfile.NoteLines) > 0 {
			noteLines = append(noteLines, existingProfile.NoteLines...)
		}

		if profileUseCase.IsEmptyProfile(noteLines, selectedTags) {
			savedProfile = nil
			return commandSvc.DeleteSubjectProfile(ctx, profileID)
		}

		profile := profileUseCase.BuildSubjectProfileModel(ownerID, request.SubjectID, noteLines, selectedTags, u.nowFn())
		if err := commandSvc.SaveSubjectProfile(ctx, profile); err != nil {
			return err
		}
		savedProfile = profile
		return nil
	}); err != nil {
		return nil, err
	}

	return profileUseCase.ToSubjectProfileResponse(savedProfile, request.SubjectID), nil
}

func (u *profileCommandUseCase) runSubjectProfileTx(
	ctx context.Context,
	fn func(context.Context, qryService.SubjectProfileQueryService, cmdService.SubjectProfileCommandService) error,
) error {
	if u.client == nil {
		return fn(ctx, u.subjectProfileQuerySvc, u.subjectProfileCommandSvc)
	}

	return u.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		return fn(
			ctx,
			u.subjectProfileQuerySvc.WithTx(tx),
			u.subjectProfileCommandSvc.WithTx(tx),
		)
	})
}
