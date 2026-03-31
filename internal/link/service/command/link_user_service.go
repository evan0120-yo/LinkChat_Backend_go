package command

import (
	"context"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/repository"
)

type LinkUserCommandService interface {
	WithTx(tx *firestore.Transaction) LinkUserCommandService
	CreateLinkUser(ctx context.Context, user *model.LinkUser) error
	UpdateLinkUser(ctx context.Context, user *model.LinkUser) error
}

type linkUserCommandService struct {
	repo repository.LinkUserRepository
}

func NewLinkUserCommandService(repo repository.LinkUserRepository) LinkUserCommandService {
	return &linkUserCommandService{repo: repo}
}

func (s *linkUserCommandService) WithTx(tx *firestore.Transaction) LinkUserCommandService {
	return &linkUserCommandService{repo: s.repo.WithTx(tx)}
}

func (s *linkUserCommandService) CreateLinkUser(ctx context.Context, user *model.LinkUser) error {
	return s.repo.CreateLinkUser(ctx, user)
}

func (s *linkUserCommandService) UpdateLinkUser(ctx context.Context, user *model.LinkUser) error {
	return s.repo.UpdateLinkUser(ctx, user)
}
