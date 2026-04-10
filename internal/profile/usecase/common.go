package usecase

import (
	"context"
	"errors"
	"time"

	linkQryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"github.com/evan0120-yo/linkchat-go/internal/profile/object/resp"
)

var (
	ErrOwnerIDRequired      = errors.New("owner id is required")
	ErrSubjectIDRequired    = errors.New("subjectId is required")
	ErrSubjectIsCurrentUser = errors.New("subjectId cannot be the current user")
	ErrSubjectNotAccessible = errors.New("subject not found or not linked")
)

func EnsureAccessibleSubject(
	ctx context.Context,
	linkQueryUseCase linkQryUseCase.LinkQueryUseCase,
	ownerID string,
	subjectID string,
) error {
	if ownerID == "" {
		return ErrOwnerIDRequired
	}
	if subjectID == "" {
		return ErrSubjectIDRequired
	}
	if ownerID == subjectID {
		return ErrSubjectIsCurrentUser
	}

	subject, err := linkQueryUseCase.GetLinkedSubject(ctx, ownerID, subjectID)
	if err != nil {
		return err
	}
	if subject == nil {
		return ErrSubjectNotAccessible
	}

	return nil
}

func BuildSubjectProfileID(ownerID, subjectID string) string {
	return ownerID + "__" + subjectID
}

func BuildSubjectProfileModel(
	ownerID string,
	subjectID string,
	noteLines []string,
	selectedTags []model.SelectedTag,
	now time.Time,
) *model.SubjectProfile {
	return &model.SubjectProfile{
		ID:           BuildSubjectProfileID(ownerID, subjectID),
		OwnerID:      ownerID,
		SubjectID:    subjectID,
		NoteLines:    noteLines,
		SelectedTags: selectedTags,
		UpdatedAt:    now.UTC(),
	}
}

func IsEmptyProfile(noteLines []string, selectedTags []model.SelectedTag) bool {
	return len(noteLines) == 0 && len(selectedTags) == 0
}

func ToSubjectProfileResponse(profile *model.SubjectProfile, subjectID string) *resp.SubjectProfileResp {
	response := &resp.SubjectProfileResp{
		SubjectID:    subjectID,
		NoteLines:    []string{},
		SelectedTags: []resp.SelectedTagResp{},
	}

	if profile == nil {
		return response
	}

	if len(profile.NoteLines) > 0 {
		response.NoteLines = append(response.NoteLines, profile.NoteLines...)
	}

	if len(profile.SelectedTags) > 0 {
		response.SelectedTags = make([]resp.SelectedTagResp, 0, len(profile.SelectedTags))
		for _, item := range profile.SelectedTags {
			response.SelectedTags = append(response.SelectedTags, resp.SelectedTagResp{
				GroupKey: item.GroupKey,
				TagKey:   item.TagKey,
			})
		}
	}

	updatedAt := profile.UpdatedAt
	response.UpdatedAt = &updatedAt

	return response
}
