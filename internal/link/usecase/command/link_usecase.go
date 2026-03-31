package command

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/google/uuid"

	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	linkService "github.com/evan0120-yo/linkchat-go/internal/link/service/command"
	linkQuery "github.com/evan0120-yo/linkchat-go/internal/link/service/query"
	linkValidator "github.com/evan0120-yo/linkchat-go/internal/link/service/validator"
)

type LinkCommandUseCase interface {
	ApplyLink(ctx context.Context, req *req.ApplyLinkReq) (*model.Link, error)
	AcceptLink(ctx context.Context, req *req.AcceptLinkReq) error
	RejectLink(ctx context.Context, req *req.RejectLinkReq) error
	RemoveLink(ctx context.Context, req *req.RemoveLinkReq) error
	// [新增]
	CancelLink(ctx context.Context, req *req.CancelLinkReq) error
}

type linkCommandUseCase struct {
	client           *firestore.Client
	validator        linkValidator.LinkValidator
	cmdService       linkService.LinkCommandService
	queryService     linkQuery.LinkQueryService
	userQueryService linkQuery.LinkUserQueryService
}

func NewLinkCommandUseCase(
	client *firestore.Client,
	validator linkValidator.LinkValidator,
	cmdService linkService.LinkCommandService,
	queryService linkQuery.LinkQueryService,
	userQueryService linkQuery.LinkUserQueryService,
) LinkCommandUseCase {
	return &linkCommandUseCase{
		client:           client,
		validator:        validator,
		cmdService:       cmdService,
		queryService:     queryService,
		userQueryService: userQueryService,
	}
}

// ApplyLink 申請好友
func (uc *linkCommandUseCase) ApplyLink(ctx context.Context, req *req.ApplyLinkReq) (*model.Link, error) {
	// 1. 驗證
	if err := uc.validator.ValidateCreateLink(req.RequesterID, req.TargetID); err != nil {
		return nil, err
	}

	var createdLink *model.Link

	// 2. Transaction
	err := uc.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		txCmd := uc.cmdService.WithTx(tx)

		// A. [Check] 檢查是否已有關係
		existingLink, err := uc.queryService.GetLinkByParticipants(ctx, req.RequesterID, req.TargetID)
		if err != nil {
			return err
		}
		if existingLink != nil {
			return errors.New("link already exists or pending")
		}

		// B. [Check] 檢查目標用戶是否存在 (避免加到空氣)
		targetUser, err := uc.userQueryService.GetLinkUserByID(ctx, req.TargetID)
		if err != nil {
			return err
		}
		if targetUser == nil || !targetUser.IsActive {
			return errors.New("target user not found or inactive")
		}

		// C. [Generate] 產生 UUID v7
		id, err := uuid.NewV7()
		if err != nil {
			return err
		}

		newLink := &model.Link{
			ID:           id.String(),
			RequesterID:  req.RequesterID,
			TargetID:     req.TargetID,
			Participants: []string{req.RequesterID, req.TargetID},
			Status:       "pending",
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		// D. [Write] 寫入
		if err := txCmd.CreateLink(ctx, newLink); err != nil {
			return err
		}

		createdLink = newLink
		return nil
	})

	if err != nil {
		return nil, err
	}

	return createdLink, nil
}

// AcceptLink 接受好友
func (uc *linkCommandUseCase) AcceptLink(ctx context.Context, req *req.AcceptLinkReq) error {
	return uc.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		txCmd := uc.cmdService.WithTx(tx)

		// 1. [Get] 獲取連結
		link, err := uc.queryService.GetLinkByID(ctx, req.LinkID)
		if err != nil {
			return err
		}
		if link == nil {
			return errors.New("link request not found")
		}

		// 2. [Validate] 安全性檢查
		if link.TargetID != req.OperatorID {
			return errors.New("permission denied: only the target user can accept this request")
		}

		// 3. [Validate] 狀態檢查
		if link.Status != "pending" {
			return errors.New("link is not in pending status")
		}

		// 4. [Modify] 修改狀態
		link.Status = "active"
		link.UpdatedAt = time.Now()

		// 5. [Update] 寫回
		return txCmd.UpdateLink(ctx, link)
	})
}

// RejectLink 拒絕好友
func (uc *linkCommandUseCase) RejectLink(ctx context.Context, req *req.RejectLinkReq) error {
	return uc.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		txCmd := uc.cmdService.WithTx(tx)

		// 1. [Get]
		link, err := uc.queryService.GetLinkByID(ctx, req.LinkID)
		if err != nil {
			return err
		}
		if link == nil {
			return errors.New("link request not found")
		}

		// 2. [Domain Logic]
		if err := txCmd.RejectLink(ctx, link, req.OperatorID); err != nil {
			return err
		}

		return nil
	})
}

// RemoveLink 解除好友
func (uc *linkCommandUseCase) RemoveLink(ctx context.Context, req *req.RemoveLinkReq) error {
	return uc.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		txCmd := uc.cmdService.WithTx(tx)

		// 1. [Get]
		link, err := uc.queryService.GetLinkByID(ctx, req.LinkID)
		if err != nil {
			return err
		}
		if link == nil {
			return errors.New("link not found")
		}

		// 2. [Domain Logic]
		if err := txCmd.RemoveLink(ctx, link, req.OperatorID); err != nil {
			return err
		}

		return nil
	})
}

// [新增] CancelLink 收回申請
func (uc *linkCommandUseCase) CancelLink(ctx context.Context, req *req.CancelLinkReq) error {
	return uc.client.RunTransaction(ctx, func(ctx context.Context, tx *firestore.Transaction) error {
		txCmd := uc.cmdService.WithTx(tx)

		// 1. [Get] 讀取 Link
		link, err := uc.queryService.GetLinkByID(ctx, req.LinkID)
		if err != nil {
			return err
		}
		if link == nil {
			return errors.New("link request not found")
		}

		// 2. [Domain Logic] 呼叫 Service 執行收回邏輯
		if err := txCmd.CancelLink(ctx, link, req.OperatorID); err != nil {
			return err
		}

		return nil
	})
}
