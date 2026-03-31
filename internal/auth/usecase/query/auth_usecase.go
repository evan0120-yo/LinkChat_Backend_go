package query

import (
	"context"
	"fmt"

	// 修改成你的 Module Name
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/object/req"
	"github.com/evan0120-yo/linkchat-go/internal/auth/object/resp"
	qryService "github.com/evan0120-yo/linkchat-go/internal/auth/service/query"
)

// Interface
type AuthQueryUseCase interface {
	Login(ctx context.Context, query *req.LoginReq) (*resp.LoginResp, error)
	FindUserByID(ctx context.Context, userID string) (*model.User, error)
}

// Implementation
type authQueryUseCase struct {
	queryService qryService.AuthQueryService
}

func NewAuthQueryUseCase(qs qryService.AuthQueryService) AuthQueryUseCase {
	return &authQueryUseCase{
		queryService: qs,
	}
}

// Login 實作 (Flow Orchestration)
func (u *authQueryUseCase) Login(ctx context.Context, query *req.LoginReq) (*resp.LoginResp, error) {
	// 1. 查找使用者 (Find)
	user, err := u.queryService.FindByEmail(ctx, query.Email)
	if err != nil {
		return nil, fmt.Errorf("find user failed: %w", err)
	}
	if user == nil {
		// 這裡做模糊錯誤處理，避免洩漏帳號是否存在
		// 但 Log 可以印詳細一點
		return nil, fmt.Errorf("invalid credentials")
	}

	// 2. 驗證密碼 (Verify)
	if err := u.queryService.VerifyPassword(user.Password, query.Password); err != nil {
		return nil, err
	}

	// 3. 產生 Token (Generate)
	token, err := u.queryService.GenerateToken(user)
	if err != nil {
		return nil, err
	}

	// 4. 轉換成 UseCase 層的 DTO (如果需要的話，這裡剛好結構一樣，直接轉接)
	return &resp.LoginResp{
		AccessToken: token.AccessToken,
		TokenType:   token.TokenType,
		ExpiresIn:   token.ExpiresIn,
	}, nil
}

// FindUserByID implements AuthQueryUseCase.
func (u *authQueryUseCase) FindUserByID(ctx context.Context, userID string) (*model.User, error) {
	user, err := u.queryService.FindUserByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user failed: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}
