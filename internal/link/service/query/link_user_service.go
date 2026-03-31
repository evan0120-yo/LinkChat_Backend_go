package query

import (
	"context"

	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/repository"
)

type LinkUserQueryService interface {
	GetLinkUserByID(ctx context.Context, id string) (*model.LinkUser, error)
	GetLinkUsersByIDs(ctx context.Context, ids []string) ([]*model.LinkUser, error)
	SearchUsers(ctx context.Context, keyword string) ([]*model.LinkUser, error)
}

type linkUserQueryService struct {
	repo repository.LinkUserRepository
}

func NewLinkUserQueryService(repo repository.LinkUserRepository) LinkUserQueryService {
	return &linkUserQueryService{repo: repo}
}

func (s *linkUserQueryService) GetLinkUserByID(ctx context.Context, id string) (*model.LinkUser, error) {
	return s.repo.FindLinkUserByID(ctx, id)
}

func (s *linkUserQueryService) GetLinkUsersByIDs(ctx context.Context, ids []string) ([]*model.LinkUser, error) {
	return s.repo.FindByIDs(ctx, ids)
}

func (s *linkUserQueryService) SearchUsers(ctx context.Context, keyword string) ([]*model.LinkUser, error) {
	if keyword == "" {
		return []*model.LinkUser{}, nil
	}
	return s.repo.SearchByDisplayName(ctx, keyword)
}
