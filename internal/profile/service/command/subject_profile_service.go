package command

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"github.com/evan0120-yo/linkchat-go/internal/profile/repository"
)

type SubjectProfileCommandService interface {
	WithTx(tx *firestore.Transaction) SubjectProfileCommandService
	SaveSubjectProfile(ctx context.Context, profile *model.SubjectProfile) error
	DeleteSubjectProfile(ctx context.Context, profileID string) error
}

type subjectProfileCommandService struct {
	repo repository.SubjectProfileRepository
}

func NewSubjectProfileCommandService(repo repository.SubjectProfileRepository) SubjectProfileCommandService {
	return &subjectProfileCommandService{repo: repo}
}

func (s *subjectProfileCommandService) WithTx(tx *firestore.Transaction) SubjectProfileCommandService {
	return &subjectProfileCommandService{repo: s.repo.WithTx(tx)}
}

func (s *subjectProfileCommandService) SaveSubjectProfile(ctx context.Context, profile *model.SubjectProfile) error {
	return s.repo.SaveSubjectProfile(ctx, profile)
}

func (s *subjectProfileCommandService) DeleteSubjectProfile(ctx context.Context, profileID string) error {
	return s.repo.DeleteSubjectProfile(ctx, profileID)
}
