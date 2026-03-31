package link

import (
	"cloud.google.com/go/firestore"

	"github.com/evan0120-yo/linkchat-go/internal/link/handler"
	"github.com/evan0120-yo/linkchat-go/internal/link/repository"
	"github.com/evan0120-yo/linkchat-go/internal/link/seeder"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/link/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/link/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/link/service/validator"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
)

// Module 封裝 Link 模組對外提供的所有依賴
// 包含: 給 Auth 用的 UseCase (Sync), 以及給 Router 用的 Handler (API), 以及 Seeder
type Module struct {
	LinkUserCommandUseCase cmdUseCase.LinkUserCommandUseCase
	Handler                *handler.LinkHandler
	Seeder                 *seeder.LinkSeeder
}

// NewLinkModule 負責初始化 Link 模組
func NewLinkModule(client *firestore.Client) *Module {

	// ==========================================
	// 1. Repositories (資料存取層)
	// ==========================================
	linkUserRepository := repository.NewLinkUserRepository(client)
	linkRepository := repository.NewLinkRepository(client)

	// ==========================================
	// 2. Services (領域服務層)
	// ==========================================
	linkValidator := validator.NewLinkValidator()

	// LinkUser Services (既有的: 處理使用者同步與查詢)
	linkUserCommandService := cmdService.NewLinkUserCommandService(linkUserRepository)
	linkUserQueryService := qryService.NewLinkUserQueryService(linkUserRepository)

	// Link Services (新增的: 處理好友關係)
	linkCommandService := cmdService.NewLinkCommandService(linkRepository)
	linkQueryService := qryService.NewLinkQueryService(linkRepository)

	// ==========================================
	// 3. UseCases (應用邏輯層)
	// ==========================================

	// A. LinkUser UseCase (提供給 Auth 模組 Sync 用)
	linkUserCommandUseCase := cmdUseCase.NewLinkUserCommandUseCase(
		linkUserCommandService,
		linkUserQueryService,
	)

	// B. Link Command UseCase (處理好友申請/接受)
	// 注意: 這裡注入了 linkUserQueryService 用於 "ApplyLink" 時檢查目標用戶是否存在
	linkCommandUseCase := cmdUseCase.NewLinkCommandUseCase(
		client,
		linkValidator,
		linkCommandService,
		linkQueryService,
		linkUserQueryService,
	)

	// C. Link Query UseCase (處理搜尋)
	linkQueryUseCase := qryUseCase.NewLinkQueryUseCase(linkUserQueryService, linkQueryService)

	// ==========================================
	// 4. Seeder (資料植入)
	// ==========================================
	// 注入剛建立好的 UseCase，讓 Seeder 可以模擬真實操作
	linkSeeder := seeder.NewLinkSeeder(client, linkCommandUseCase, linkQueryUseCase)

	// ==========================================
	// 5. Handler (介面層)
	// ==========================================
	linkHandler := handler.NewLinkHandler(linkCommandUseCase, linkQueryUseCase)

	// 回傳 Module 結構，讓 Main 可以分別取得需要的依賴
	return &Module{
		LinkUserCommandUseCase: linkUserCommandUseCase,
		Handler:                linkHandler,
		Seeder:                 linkSeeder,
	}
}
