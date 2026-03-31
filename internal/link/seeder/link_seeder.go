package seeder

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"

	"github.com/evan0120-yo/linkchat-go/internal/link/model"
	"github.com/evan0120-yo/linkchat-go/internal/link/object/req"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
)

type LinkSeeder struct {
	client     *firestore.Client
	cmdUseCase cmdUseCase.LinkCommandUseCase
	qryUseCase qryUseCase.LinkQueryUseCase
}

// NewLinkSeeder 需要同時依賴 Command (做動作) 與 Query (找人) UseCase
func NewLinkSeeder(
	client *firestore.Client,
	cmdUC cmdUseCase.LinkCommandUseCase,
	qryUC qryUseCase.LinkQueryUseCase,
) *LinkSeeder {
	return &LinkSeeder{
		client:     client,
		cmdUseCase: cmdUC,
		qryUseCase: qryUC,
	}
}

// Seed 負責製造測試用的好友關係
func (s *LinkSeeder) Seed(ctx context.Context) error {
	fmt.Println("[LinkSeeder] 正在植入好友關係資料...")

	// ==========================================
	// 步驟 1: 找出關鍵人物 (ID Resolution)
	// ==========================================
	// 因為 AuthSeeder 產生的 UUID 是隨機的，我們不知道 ID 是什麼
	// 所以我們透過 DisplayName 把他們找出來
	// 目標: "Normal User" (Requester) -> "EvanHe" (Target)

	// 1-1. 找 requester (Normal User)
	requester, err := s.findUserByName(ctx, "Normal User")
	if err != nil {
		fmt.Printf("   [Skip] 無法找到申請人 'Normal User': %v\n", err)
		return nil // 不中斷，可能是 AuthSeeder 還沒跑或資料被改了
	}

	// 1-2. 找 target (EvanHe)
	target, err := s.findUserByName(ctx, "EvanHe")
	if err != nil {
		fmt.Printf("   [Skip] 無法找到目標 'EvanHe': %v\n", err)
		return nil
	}

	fmt.Printf("   [解析成功] 申請人: %s (%s) -> 目標: %s (%s)\n",
		requester.DisplayName, requester.ID, target.DisplayName, target.ID)

	// ==========================================
	// 步驟 2: 建立申請 (Pending 狀態)
	// ==========================================
	applyReq := &req.ApplyLinkReq{
		RequesterID: requester.ID,
		TargetID:    target.ID,
	}

	// 呼叫 UseCase 執行申請
	// 這會同時測試: Validator -> Transaction -> Check Existence -> Create Link
	link, err := s.cmdUseCase.ApplyLink(ctx, applyReq)
	if err != nil {
		// 處理冪等性：如果已經申請過了，我們就當作成功，不要報錯
		// 錯誤訊息可能包含 "link already exists" 或 "pending"
		errMsg := strings.ToLower(err.Error())
		if strings.Contains(errMsg, "exists") || strings.Contains(errMsg, "pending") || strings.Contains(errMsg, "active") {
			fmt.Println("   關係已存在，略過建立。")
			return nil
		}

		// 如果是其他錯誤 (例如 DB 連不上)，才回傳 Error
		return fmt.Errorf("failed to apply link: %w", err)
	}

	fmt.Printf("   [建立成功] LinkID: %s | Status: %s\n", link.ID, link.Status)
	fmt.Println("      現在你可以登入 EvanHe 的帳號，在 '待確認列表' 中看到這筆申請了。")

	return nil
}

// Helper: 透過名字找人的輔助函式
func (s *LinkSeeder) findUserByName(ctx context.Context, name string) (*model.LinkUser, error) {
	// 使用 UseCase 的 SearchUsers 功能
	users, err := s.qryUseCase.SearchUsers(ctx, name)
	if err != nil {
		return nil, err
	}

	// 簡單過濾：找完全符合 DisplayName 的人
	for _, u := range users {
		if u.DisplayName == name {
			return u, nil
		}
	}

	return nil, fmt.Errorf("user not found with name: %s", name)
}
