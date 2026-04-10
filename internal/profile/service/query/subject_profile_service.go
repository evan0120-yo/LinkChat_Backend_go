package query

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"github.com/evan0120-yo/linkchat-go/internal/profile/repository"
)

type SubjectProfileQueryService interface {
	WithTx(tx *firestore.Transaction) SubjectProfileQueryService
	GetSubjectProfileByID(ctx context.Context, profileID string) (*model.SubjectProfile, error)
}

type subjectProfileQueryService struct {
	repo repository.SubjectProfileRepository
}

func NewSubjectProfileQueryService(repo repository.SubjectProfileRepository) SubjectProfileQueryService {
	return &subjectProfileQueryService{repo: repo}
}

func (s *subjectProfileQueryService) WithTx(tx *firestore.Transaction) SubjectProfileQueryService {
	return &subjectProfileQueryService{repo: s.repo.WithTx(tx)}
}

func (s *subjectProfileQueryService) GetSubjectProfileByID(ctx context.Context, profileID string) (*model.SubjectProfile, error) {
	return s.repo.FindSubjectProfileByID(ctx, profileID)
}
