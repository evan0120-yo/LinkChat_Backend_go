package seeder

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"

	"github.com/evan0120-yo/linkchat-go/internal/auth/model"
	"github.com/evan0120-yo/linkchat-go/internal/auth/object/req" // 引用 Request Object

	// 改成依賴 UseCase
	authUseCase "github.com/evan0120-yo/linkchat-go/internal/auth/usecase/command"
)

type AuthSeeder struct {
	client  *firestore.Client
	useCase authUseCase.AuthCommandUseCase // 依賴 UseCase
}

// 修改 Factory
func NewAuthSeeder(client *firestore.Client, uc authUseCase.AuthCommandUseCase) *AuthSeeder {
	return &AuthSeeder{
		client:  client,
		useCase: uc,
	}
}

// Seed 負責寫入所有初始資料
func (s *AuthSeeder) Seed(ctx context.Context) error {
	fmt.Println("[Seeder] 正在植入初始帳號資料 (透過 UseCase)...")

	// 1. 定義要植入的資料
	seedUsers := []struct {
		Email       string
		Password    string
		DisplayName string
		Role        model.Role
	}{
		{
			Email:       "admin@linkchat.com",
			Password:    "admin123",
			DisplayName: "System Admin",
			Role:        model.RoleAdmin,
		},
		{
			Email:       "user@linkchat.com",
			Password:    "user123",
			DisplayName: "Normal User",
			Role:        model.RoleUser,
		},
		{
			Email:       "evan01203394@gmail.com",
			Password:    "evan",
			DisplayName: "EvanHe",
			Role:        model.RoleUser,
		},
	}

	// 2. 迴圈呼叫 UseCase (不用自己開 Transaction，UseCase 裡有)
	for _, seedData := range seedUsers {
		// 組裝 Register Request
		regReq := &req.RegisterReq{
			Email:       seedData.Email,
			Password:    seedData.Password, // 傳入原始密碼，UseCase 會幫忙 Hash
			DisplayName: seedData.DisplayName,
		}

		// 呼叫 UseCase.Register
		// 這會自動完成: Hash Password -> Create User -> Sync to Link Module
		err := s.useCase.Register(ctx, regReq)

		if err != nil {
			// 如果已經存在 (AlreadyExists)，我們就略過，這樣 Seeder 可以重複跑
			// 這裡簡單印個 Log 就好
			fmt.Printf("   帳號已存在或建立失敗: %s (%v)\n", seedData.Email, err)
			continue
		}

		fmt.Printf("   建立成功: %s\n", seedData.DisplayName)

		// 注意：標準的 Register UseCase 通常會強制設定 Role 為 "user"。
		// 如果你需要讓 Admin 真的變成 Admin 權限，
		// 你可能需要在這裡額外呼叫一個 "ForceUpdateRole" 的 Service，
		// 或是修改 RegisterReq 讓他支援後門 (但不建議)。
		// 為了 MVP，我們先假設 Register 都創成 User 沒關係，
		// 或者你可以手動去 Firestore 改 Admin 的 Role。
	}

	fmt.Println("初始資料植入完成")
	return nil
}
