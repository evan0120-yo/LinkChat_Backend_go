package auth

import (
	"cloud.google.com/go/firestore"

	// Auth 模組
	"github.com/evan0120-yo/linkchat-go/internal/auth/handler"
	"github.com/evan0120-yo/linkchat-go/internal/auth/middleware"
	authRepository "github.com/evan0120-yo/linkchat-go/internal/auth/repository"
	"github.com/evan0120-yo/linkchat-go/internal/auth/seeder"
	authCommandService "github.com/evan0120-yo/linkchat-go/internal/auth/service/command"
	authQueryService "github.com/evan0120-yo/linkchat-go/internal/auth/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/auth/service/validator"
	authCommandUseCase "github.com/evan0120-yo/linkchat-go/internal/auth/usecase/command"
	authQueryUseCase "github.com/evan0120-yo/linkchat-go/internal/auth/usecase/query"

	// 引用 Link UseCase (作為依賴介面)
	linkUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/command"
)

// NewAuthModule 負責組裝 Auth 模組的所有元件
// 修正：多了一個參數 linkUserCmdUseCase (Interface)
func NewAuthModule(
	client *firestore.Client,
	linkUserCmdUseCase linkUseCase.LinkUserCommandUseCase,
) (
	*handler.AuthHandler,
	*seeder.AuthSeeder,
	*middleware.AuthMiddleware,
	*handler.TestHandler,
) {
	// ==========================================
	// 初始化 Auth 模組元件
	// ==========================================

	// Repo
	userRepo := authRepository.NewUserRepository(client)

	// Service
	authQryService := authQueryService.NewAuthQueryService(userRepo)
	authCmdService := authCommandService.NewAuthCommandService(userRepo)

	// Validator
	authValidator := validator.NewAuthValidator(authQryService)

	// UseCase
	authQryUseCase := authQueryUseCase.NewAuthQueryUseCase(authQryService)

	// Command UseCase (注入 Link UseCase)
	authCmdUseCase := authCommandUseCase.NewAuthCommandUseCase(
		client,
		authCmdService,
		authQryService,
		authValidator,
		linkUserCmdUseCase, // 這裡直接塞入傳進來的 Interface
	)

	// Handler
	authHandler := handler.NewAuthHandler(authCmdUseCase, authQryUseCase)
	testHandler := handler.NewTestHandler()

	// Seeder
	authSeeder := seeder.NewAuthSeeder(client, authCmdUseCase)

	// Middleware
	jwtSecret := "YOUR_SUPER_SECRET_KEY" // 記得改
	authMiddleware := middleware.NewAuthMiddleware(jwtSecret)

	return authHandler, authSeeder, authMiddleware, testHandler
}
