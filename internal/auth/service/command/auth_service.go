package command

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/repository"
	"golang.org/x/crypto/bcrypt"
)

// Interface
type AuthCommandService interface {
	// WithTx 綁定交易
	WithTx(tx *firestore.Transaction) AuthCommandService
	HashPassword(password string) (string, error)
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error
}

// Implementation
type authCommandService struct {
	repo repository.UserRepository
}

func NewAuthCommandService(repo repository.UserRepository) AuthCommandService {
	return &authCommandService{repo: repo}
}

// WithTx 實作
func (s *authCommandService) WithTx(tx *firestore.Transaction) AuthCommandService {
	return &authCommandService{repo: s.repo.WithTx(tx)}
}

// HashPassword 實作
func (s *authCommandService) HashPassword(password string) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password failed: %w", err)
	}
	return string(hashedPwd), nil
}

// CreateUser 實作
func (s *authCommandService) CreateUser(ctx context.Context, user *model.User) error {
	return s.repo.CreateUser(ctx, user)
}

// UpdateUser 實作
func (s *authCommandService) UpdateUser(ctx context.Context, user *model.User) error {
	return s.repo.UpdateUser(ctx, user)
}
