package command

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"

	// 修改成你的 Module Name
	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/object/req"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/auth/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/auth/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/auth/service/validator"
	linkReq "github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	linkUserUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/command"
)

// Interface
type AuthCommandUseCase interface {
	Register(ctx context.Context, cmd *req.RegisterReq) error
	DeleteUser(ctx context.Context, actorID string, actorRole model.Role, targetUserID string) error
}

// Implementation
type authCommandUseCase struct {
	client                 *firestore.Client             // 負責開啟 Transaction
	authCommandService     cmdService.AuthCommandService // 負責寫入動作
	authQueryService       qryService.AuthQueryService   // 負責查詢動作
	authValidator          validator.AuthValidator       // 負責驗證
	linkUserCommandUseCase linkUserUseCase.LinkUserCommandUseCase
}

func NewAuthCommandUseCase(
	client *firestore.Client,
	authCommandService cmdService.AuthCommandService,
	authQueryService qryService.AuthQueryService,
	authValidator validator.AuthValidator,
	linkUserCommandUseCase linkUserUseCase.LinkUserCommandUseCase,
) AuthCommandUseCase {
	return &authCommandUseCase{
		client:                 client,
		authCommandService:     authCommandService,
		authQueryService:       authQueryService,
		authValidator:          authValidator,
		linkUserCommandUseCase: linkUserCommandUseCase,
	}
}

func (u *authCommandUseCase) Register(ctx context.Context, cmd *req.RegisterReq) error {
	// 1. 驗證邏輯 (Validator)
	// Query 可以直接 Call，這裡呼叫 Validator 去查 DB 有沒有重複
	if err := u.authValidator.ValidateEmailUnique(ctx, cmd.Email); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// 2. 密碼加密 (Command Service Helper)
	hashedPwd, err := u.authCommandService.HashPassword(cmd.Password)
	if err != nil {
		return err
	}

	// 3. 生成 ID (UseCase 決定 ID)
	uuidV7, err := uuid.NewV7()
	if err != nil {
		return fmt.Errorf("failed to generate uuid: %w", err)
	}

	// 4. 組裝 Domain Model
	newUser := &model.User{
		ID:          uuidV7.String(),
		Email:       cmd.Email,
		Password:    hashedPwd,
		DisplayName: cmd.DisplayName,
		Role:        model.RoleUser,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}

	// 5. 交易提交 (Transaction Commit)
	// 預設裡面還會直接call 各個module新增用戶
	err = u.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// 綁定 Tx 到 Command Service
		txAuthCommandService := u.authCommandService.WithTx(tx)

		if err := txAuthCommandService.CreateUser(ctx, newUser); err != nil {
			return err
		}

		// sync link user
		if err := u.linkUserCommandUseCase.SyncUser(ctx, tx, &linkReq.CreateLinkUserReq{
			ID:          newUser.ID,
			DisplayName: newUser.DisplayName,
		}); err != nil {
			return err
		}

		// return nil -> 自動 Commit
		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}

// Implementation 修改
func (u *authCommandUseCase) DeleteUser(ctx context.Context, actorID string, actorRole model.Role, targetUserID string) error {
	fmt.Println(actorID, actorRole, targetUserID)

	// ==========================================
	// 1. 權限檢查 (Business Rule)
	// ==========================================
	// 規則：如果 "操作者不是 Admin" 或 "操作者 ID 不等於 目標 ID" -> 拒絕
	if actorRole != model.RoleAdmin && actorID != targetUserID {
		return fmt.Errorf("permission denied: you can only delete yourself")
	}

	// ==========================================
	// 2. 檢查目標用戶是否存在
	// ==========================================
	user, err := u.authQueryService.FindUserByID(ctx, targetUserID)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("user not found")
	}

	// ==========================================
	// 3. 改狀態成 false
	// ==========================================
	user.IsActive = false

	// ==========================================
	// 4. Transaction (Auth Update + Link Delete)
	// ==========================================
	err = u.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		// 綁定 Tx 到 Command Service
		txCmdService := u.authCommandService.WithTx(tx)

		if err := txCmdService.UpdateUser(ctx, user); err != nil {
			return err
		}

		// 同步刪除 Link 模組資料
		err = u.linkUserCommandUseCase.DeleteLinkUser(ctx, tx, targetUserID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}
	return nil
}
