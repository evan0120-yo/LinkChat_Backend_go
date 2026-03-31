package query

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Interface
type AuthQueryService interface {
	FindByEmail(ctx context.Context, email string) (*model.User, error)
	VerifyPassword(hashedPassword, inputPassword string) error
	GenerateToken(user *model.User) (*model.AuthToken, error)
	FindUserByID(ctx context.Context, userID string) (*model.User, error)
}

// Implementation
type authQueryService struct {
	repo      repository.UserRepository
	jwtSecret []byte
}

func NewAuthQueryService(repo repository.UserRepository) AuthQueryService {
	return &authQueryService{
		repo:      repo,
		jwtSecret: []byte("YOUR_SUPER_SECRET_KEY"), // TODO: Config
	}
}

func (s *authQueryService) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	return s.repo.FindByEmail(ctx, email)
}

func (s *authQueryService) FindUserByID(ctx context.Context, userID string) (*model.User, error) {
	return s.repo.FindByID(ctx, userID)
}

func (s *authQueryService) VerifyPassword(hashedPassword, inputPassword string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	if err != nil {
		return errors.New("invalid credentials")
	}
	return nil
}

func (s *authQueryService) GenerateToken(user *model.User) (*model.AuthToken, error) {
	// 設定 Claims
	claims := jwt.MapClaims{
		"sub":  user.ID,
		"name": user.DisplayName,
		"role": user.Role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iss":  "linkchat-backend",
	}

	// 簽署 Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("generate token failed: %w", err)
	}

	return &model.AuthToken{
		AccessToken: tokenString,
		TokenType:   "Bearer",
		ExpiresIn:   86400,
	}, nil
}
