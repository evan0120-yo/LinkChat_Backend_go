package profile

import (
	"cloud.google.com/go/firestore"

	linkQryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/handler"
	"github.com/evan0120-yo/linkchat-go/internal/profile/repository"
	"github.com/evan0120-yo/linkchat-go/internal/profile/seeder"
	cmdService "github.com/evan0120-yo/linkchat-go/internal/profile/service/command"
	qryService "github.com/evan0120-yo/linkchat-go/internal/profile/service/query"
	"github.com/evan0120-yo/linkchat-go/internal/profile/service/validator"
	cmdUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/command"
	qryUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase/query"
)

type Module struct {
	Handler *handler.ProfileHandler
	Seeder  *seeder.ProfileSeeder
}

func NewProfileModule(
	client *firestore.Client,
	linkQueryUseCase linkQryUseCase.LinkQueryUseCase,
) *Module {
	subjectProfileRepository := repository.NewSubjectProfileRepository(client)
	tagCatalogRepository := repository.NewTagCatalogRepository(client)

	subjectProfileCommandService := cmdService.NewSubjectProfileCommandService(subjectProfileRepository)
	subjectProfileQueryService := qryService.NewSubjectProfileQueryService(subjectProfileRepository)
	tagCatalogQueryService := qryService.NewTagCatalogQueryService(tagCatalogRepository)
	profileValidator := validator.NewProfileValidator()

	profileCommandUseCase := cmdUseCase.NewProfileCommandUseCase(
		client,
		profileValidator,
		subjectProfileCommandService,
		subjectProfileQueryService,
		tagCatalogQueryService,
		linkQueryUseCase,
	)
	profileQueryUseCase := qryUseCase.NewProfileQueryUseCase(
		subjectProfileQueryService,
		tagCatalogQueryService,
		linkQueryUseCase,
	)

	return &Module{
		Handler: handler.NewProfileHandler(profileCommandUseCase, profileQueryUseCase),
		Seeder:  seeder.NewProfileSeeder(tagCatalogRepository),
	}
}
