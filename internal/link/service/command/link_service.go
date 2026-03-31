package command

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/repository"
)

type LinkCommandService interface {
	WithTx(tx *firestore.Transaction) LinkCommandService
	CreateLink(ctx context.Context, link *model.Link) error
	UpdateLink(ctx context.Context, link *model.Link) error
	RejectLink(ctx context.Context, link *model.Link, operatorID string) error
	RemoveLink(ctx context.Context, link *model.Link, operatorID string) error
	// [新增] 收回申請
	CancelLink(ctx context.Context, link *model.Link, operatorID string) error
}

type linkCommandService struct {
	repo repository.LinkRepository
}

func NewLinkCommandService(repo repository.LinkRepository) LinkCommandService {
	return &linkCommandService{repo: repo}
}

func (s *linkCommandService) WithTx(tx *firestore.Transaction) LinkCommandService {
	return &linkCommandService{repo: s.repo.WithTx(tx)}
}

func (s *linkCommandService) CreateLink(ctx context.Context, link *model.Link) error {
	return s.repo.CreateLink(ctx, link)
}

func (s *linkCommandService) UpdateLink(ctx context.Context, link *model.Link) error {
	return s.repo.UpdateLink(ctx, link)
}

func (s *linkCommandService) RejectLink(ctx context.Context, link *model.Link, operatorID string) error {
	// 1. 驗證權限：只有「被申請人 (Target)」有資格拒絕
	if link.TargetID != operatorID {
		return errors.New("permission denied: only target user can reject")
	}

	// 2. 驗證狀態：必須是 "pending" 才能拒絕
	if link.Status != "pending" {
		return errors.New("invalid status: can only reject pending requests")
	}

	// 3. 執行拒絕 (更新狀態)
	link.Status = "rejected"
	link.UpdatedAt = time.Now()

	return s.repo.UpdateLink(ctx, link)
}

func (s *linkCommandService) RemoveLink(ctx context.Context, link *model.Link, operatorID string) error {
	// 1. 驗證權限：操作者必須是該關係的參與者之一
	isParticipant := false
	for _, p := range link.Participants {
		if p == operatorID {
			isParticipant = true
			break
		}
	}
	if !isParticipant {
		return errors.New("permission denied: you are not a participant of this link")
	}

	// 2. 驗證狀態：通常只能解除 "active" 的好友關係
	if link.Status != "active" {
		return errors.New("invalid status: can only remove active friends")
	}

	// 3. 執行刪除 (Hard Delete)
	return s.repo.DeleteLink(ctx, link.ID)
}

// [新增] CancelLink 實作
func (s *linkCommandService) CancelLink(ctx context.Context, link *model.Link, operatorID string) error {
	// 1. 驗證權限：只有「申請人 (Requester)」可以收回
	if link.RequesterID != operatorID {
		return errors.New("permission denied: only the requester can cancel this request")
	}

	// 2. 驗證狀態：必須是 "pending" 才能收回
	if link.Status != "pending" {
		return errors.New("invalid status: can only cancel pending requests")
	}

	// 3. 執行刪除 (Hard Delete)
	return s.repo.DeleteLink(ctx, link.ID)
}
