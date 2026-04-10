package query

import (
	"context"

	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"github.com/evan0120-yo/linkchat-go/internal/profile/repository"
)

type TagCatalogQueryService interface {
	GetActiveTagCatalog(ctx context.Context) (*model.TagCatalog, error)
}

type tagCatalogQueryService struct {
	repo repository.TagCatalogRepository
}

func NewTagCatalogQueryService(repo repository.TagCatalogRepository) TagCatalogQueryService {
	return &tagCatalogQueryService{repo: repo}
}

func (s *tagCatalogQueryService) GetActiveTagCatalog(ctx context.Context) (*model.TagCatalog, error) {
	groups, err := s.repo.ListActiveGroups(ctx)
	if err != nil {
		return nil, err
	}

	tags, err := s.repo.ListActiveTags(ctx)
	if err != nil {
		return nil, err
	}

	return &model.TagCatalog{
		Groups: groups,
		Tags:   tags,
	}, nil
}
