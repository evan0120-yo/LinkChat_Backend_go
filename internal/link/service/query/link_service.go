package query

import (
	"context"

	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/repository"
)

type LinkQueryService interface {
	GetLinkByID(ctx context.Context, id string) (*model.Link, error)
	GetLinksByUserID(ctx context.Context, userID string) ([]*model.Link, error)

	// GetLinkByParticipants 用於檢查兩者是否已有關係 (Query-First Check)
	GetLinkByParticipants(ctx context.Context, userA, userB string) (*model.Link, error)
}

type linkQueryService struct {
	repo repository.LinkRepository
}

func NewLinkQueryService(repo repository.LinkRepository) LinkQueryService {
	return &linkQueryService{repo: repo}
}

func (s *linkQueryService) GetLinkByID(ctx context.Context, id string) (*model.Link, error) {
	return s.repo.FindLinkByID(ctx, id)
}

func (s *linkQueryService) GetLinksByUserID(ctx context.Context, userID string) ([]*model.Link, error) {
	return s.repo.FindLinksByUserID(ctx, userID)
}

func (s *linkQueryService) GetLinkByParticipants(ctx context.Context, userA, userB string) (*model.Link, error) {
	return s.repo.FindLinkByParticipants(ctx, userA, userB)
}
