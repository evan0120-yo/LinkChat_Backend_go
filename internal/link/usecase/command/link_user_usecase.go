package command

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	linkReq "github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/link/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/link/service/query"
)

type LinkUserCommandUseCase interface {
	SyncUser(ctx context.Context, tx *firestore.Transaction, dto *linkReq.CreateLinkUserReq) error
	DeleteLinkUser(ctx context.Context, tx *firestore.Transaction, userID string) error
}

type linkUserCommandUseCase struct {
	linkUserCommandService cmdService.LinkUserCommandService
	linkUserQueryService   qryService.LinkUserQueryService
}

func NewLinkUserCommandUseCase(
	linkUserCommandService cmdService.LinkUserCommandService,
	linkUserQueryService qryService.LinkUserQueryService,
) LinkUserCommandUseCase {
	return &linkUserCommandUseCase{
		linkUserCommandService: linkUserCommandService,
		linkUserQueryService:   linkUserQueryService,
	}
}

func (u *linkUserCommandUseCase) SyncUser(ctx context.Context, tx *firestore.Transaction, dto *linkReq.CreateLinkUserReq) error {
	txService := u.linkUserCommandService.WithTx(tx)

	linkUser := &model.LinkUser{
		ID:          dto.ID,
		DisplayName: dto.DisplayName,
		IsActive:    true, // 同步過來通常代表活著
		UpdatedAt:   time.Now(),
	}

	if err := txService.CreateLinkUser(ctx, linkUser); err != nil {
		return fmt.Errorf("sync link user failed: %w", err)
	}
	return nil
}

func (u *linkUserCommandUseCase) DeleteLinkUser(ctx context.Context, tx *firestore.Transaction, userID string) error {
	// 1. [Find] 先查出資料
	linkUser, err := u.linkUserQueryService.GetLinkUserByID(ctx, userID)
	if err != nil {
		// 這裡是指 "資料庫連線錯誤" 等系統問題，必須回傳錯誤
		return fmt.Errorf("find link user failed: %w", err)
	}

	// 2. [Check] 如果找不到資料 -> 視為成功
	if linkUser == nil {
		return nil
	}

	// 3. [Modify] 修改狀態
	linkUser.IsActive = false
	linkUser.UpdatedAt = time.Now()

	// 4. [Save] 透過交易寫回
	txService := u.linkUserCommandService.WithTx(tx)

	if err := txService.UpdateLinkUser(ctx, linkUser); err != nil {
		return fmt.Errorf("delete link user failed: %w", err)
	}

	return nil
}
