package validator

import (
	"context"
	"errors"

	"github.com/evan0120-yo/linkchat-go/internal/auth/service/query"
)

// AuthValidator 定義所有驗證規則
type AuthValidator interface {
	// ValidateEmailUnique 檢查 Email 是否唯一 (註冊用)
	ValidateEmailUnique(ctx context.Context, email string) error
	// 未來如果有 Password 強度檢查、Username 檢查都可以加在這裡
}

type authValidator struct {
	queryService query.AuthQueryService
}

func NewAuthValidator(qs query.AuthQueryService) AuthValidator {
	return &authValidator{
		queryService: qs,
	}
}

// ValidateEmailUnique 實作
func (v *authValidator) ValidateEmailUnique(ctx context.Context, email string) error {
	// 呼叫 Query Service 去查
	user, err := v.queryService.FindByEmail(ctx, email)

	// 如果 err 不是 nil，代表查詢過程有錯 (DB 連線失敗等)，直接回傳錯誤
	if err != nil {
		return err
	}

	// 如果 user 不是 nil，代表資料庫裡已經有這個人 -> 驗證失敗 (重複註冊)
	if user != nil {
		return errors.New("email already exists")
	}

	// 通過驗證
	return nil
}
