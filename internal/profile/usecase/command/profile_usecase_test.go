package command

import (
	"context"
	"testing"
	"time"

	"cloud.google.com/go/firestore"
	linkModel "github.com/evan0120-yo/linkchat-go/internal/link/model"
	linkReq "github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	linkResp "github.com/evan0120-yo/linkchat-go/internal/link/object/resp"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/profile/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/profile/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/service/validator"
)

func TestSaveSubjectNotesPreservesExistingTags(t *testing.T) {
	querySvc := &fakeSubjectProfileQueryService{
		profile: &model.SubjectProfile{
			ID:        "owner-1__subject-1",
			OwnerID:   "owner-1",
			SubjectID: "subject-1",
			SelectedTags: []model.SelectedTag{
				{GroupKey: "role", TagKey: "student"},
			},
		},
	}
	commandSvc := &fakeSubjectProfileCommandService{}
	usecase := &profileCommandUseCase{
		validator:                validator.NewProfileValidator(),
		subjectProfileCommandSvc: commandSvc,
		subjectProfileQuerySvc:   querySvc,
		tagCatalogQuerySvc:       &fakeTagCatalogQueryService{},
		linkQueryUseCase:         &fakeLinkQueryUseCase{linkedSubject: &linkModel.LinkUser{ID: "subject-1", IsActive: true}},
		nowFn: func() time.Time {
			return time.Date(2026, 3, 31, 15, 0, 0, 0, time.UTC)
		},
	}

	response, err := usecase.SaveSubjectNotes(context.Background(), "owner-1", &reqObj.SaveSubjectNotesReq{
		SubjectID: "subject-1",
		Lines:     []string{"  note 1  ", "", "note 2"},
	})
	if err != nil {
		t.Fatalf("SaveSubjectNotes returned error: %v", err)
	}

	if commandSvc.savedProfile == nil {
		t.Fatal("expected profile to be saved")
	}
	if len(commandSvc.savedProfile.SelectedTags) != 1 {
		t.Fatalf("expected existing tags to be preserved, got %#v", commandSvc.savedProfile.SelectedTags)
	}
	if len(commandSvc.savedProfile.NoteLines) != 2 {
		t.Fatalf("expected 2 normalized note lines, got %#v", commandSvc.savedProfile.NoteLines)
	}
	if response.SubjectID != "subject-1" {
		t.Fatalf("unexpected subject id: %s", response.SubjectID)
	}
}

func TestSaveSubjectTagsDeletesEmptyProfile(t *testing.T) {
	querySvc := &fakeSubjectProfileQueryService{
		profile: &model.SubjectProfile{
			ID:        "owner-1__subject-1",
			OwnerID:   "owner-1",
			SubjectID: "subject-1",
		},
	}
	commandSvc := &fakeSubjectProfileCommandService{}
	usecase := &profileCommandUseCase{
		validator:                validator.NewProfileValidator(),
		subjectProfileCommandSvc: commandSvc,
		subjectProfileQuerySvc:   querySvc,
		tagCatalogQuerySvc: &fakeTagCatalogQueryService{
			catalog: &model.TagCatalog{
				Groups: []*model.TagGroup{{GroupKey: "role", SelectionMode: model.TagSelectionModeSingle}},
				Tags:   []*model.TagDefinition{{GroupKey: "role", TagKey: "student"}},
			},
		},
		linkQueryUseCase: &fakeLinkQueryUseCase{linkedSubject: &linkModel.LinkUser{ID: "subject-1", IsActive: true}},
		nowFn:            time.Now,
	}

	response, err := usecase.SaveSubjectTags(context.Background(), "owner-1", &reqObj.SaveSubjectTagsReq{
		SubjectID: "subject-1",
		Selected:  []reqObj.SelectedTagReq{},
	})
	if err != nil {
		t.Fatalf("SaveSubjectTags returned error: %v", err)
	}

	if commandSvc.deletedProfileID != "owner-1__subject-1" {
		t.Fatalf("expected profile delete, got %s", commandSvc.deletedProfileID)
	}
	if len(response.NoteLines) != 0 || len(response.SelectedTags) != 0 {
		t.Fatalf("expected empty response after delete, got %#v", response)
	}
}

func TestSaveSubjectNotesRejectsUnlinkedSubject(t *testing.T) {
	usecase := &profileCommandUseCase{
		validator:                validator.NewProfileValidator(),
		subjectProfileCommandSvc: &fakeSubjectProfileCommandService{},
		subjectProfileQuerySvc:   &fakeSubjectProfileQueryService{},
		tagCatalogQuerySvc:       &fakeTagCatalogQueryService{},
		linkQueryUseCase:         &fakeLinkQueryUseCase{},
		nowFn:                    time.Now,
	}

	_, err := usecase.SaveSubjectNotes(context.Background(), "owner-1", &reqObj.SaveSubjectNotesReq{
		SubjectID: "subject-2",
		Lines:     []string{"note"},
	})
	if err == nil {
		t.Fatal("expected error for inaccessible subject")
	}
}

type fakeSubjectProfileCommandService struct {
	savedProfile     *model.SubjectProfile
	deletedProfileID string
}

func (f *fakeSubjectProfileCommandService) WithTx(_ *firestore.Transaction) cmdService.SubjectProfileCommandService {
	return f
}

func (f *fakeSubjectProfileCommandService) SaveSubjectProfile(_ context.Context, profile *model.SubjectProfile) error {
	f.savedProfile = profile
	return nil
}

func (f *fakeSubjectProfileCommandService) DeleteSubjectProfile(_ context.Context, profileID string) error {
	f.deletedProfileID = profileID
	return nil
}

type fakeSubjectProfileQueryService struct {
	profile *model.SubjectProfile
}

func (f *fakeSubjectProfileQueryService) WithTx(_ *firestore.Transaction) qryService.SubjectProfileQueryService {
	return f
}

func (f *fakeSubjectProfileQueryService) GetSubjectProfileByID(_ context.Context, _ string) (*model.SubjectProfile, error) {
	return f.profile, nil
}

type fakeTagCatalogQueryService struct {
	catalog *model.TagCatalog
}

func (f *fakeTagCatalogQueryService) GetActiveTagCatalog(_ context.Context) (*model.TagCatalog, error) {
	if f.catalog == nil {
		return &model.TagCatalog{}, nil
	}
	return f.catalog, nil
}

type fakeLinkQueryUseCase struct {
	linkedSubject *linkModel.LinkUser
}

func (f *fakeLinkQueryUseCase) SearchUsers(context.Context, string) ([]*linkModel.LinkUser, error) {
	return nil, nil
}

func (f *fakeLinkQueryUseCase) GetLinkList(context.Context, string, linkReq.ListLinkReq) ([]*linkResp.LinkItemResp, error) {
	return nil, nil
}

func (f *fakeLinkQueryUseCase) GetLinkedSubject(context.Context, string, string) (*linkModel.LinkUser, error) {
	return f.linkedSubject, nil
}

var (
	_ cmdService.SubjectProfileCommandService = (*fakeSubjectProfileCommandService)(nil)
	_ qryService.SubjectProfileQueryService   = (*fakeSubjectProfileQueryService)(nil)
)
