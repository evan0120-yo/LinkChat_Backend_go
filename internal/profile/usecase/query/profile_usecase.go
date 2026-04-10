package query

import (
	"context"

	linkQryUseCase "github.com/evan0120-yo/linkchat-go/internal/link/usecase/query"
	respObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/resp"
	qryService "github.com/evan0120-yo/linkchat-go/internal/profile/service/query"
	profileUseCase "github.com/evan0120-yo/linkchat-go/internal/profile/usecase"
)

type ProfileQueryUseCase interface {
	GetSubjectProfileContext(ctx context.Context, ownerID string, subjectID string) (*respObj.SubjectProfileResp, error)
	GetTagCatalog(ctx context.Context) (*respObj.TagCatalogResp, error)
}

type profileQueryUseCase struct {
	subjectProfileQuerySvc qryService.SubjectProfileQueryService
	tagCatalogQuerySvc     qryService.TagCatalogQueryService
	linkQueryUseCase       linkQryUseCase.LinkQueryUseCase
}

func NewProfileQueryUseCase(
	subjectProfileQuerySvc qryService.SubjectProfileQueryService,
	tagCatalogQuerySvc qryService.TagCatalogQueryService,
	linkQueryUseCase linkQryUseCase.LinkQueryUseCase,
) ProfileQueryUseCase {
	return &profileQueryUseCase{
		subjectProfileQuerySvc: subjectProfileQuerySvc,
		tagCatalogQuerySvc:     tagCatalogQuerySvc,
		linkQueryUseCase:       linkQueryUseCase,
	}
}

func (u *profileQueryUseCase) GetSubjectProfileContext(ctx context.Context, ownerID string, subjectID string) (*respObj.SubjectProfileResp, error) {
	if err := profileUseCase.EnsureAccessibleSubject(ctx, u.linkQueryUseCase, ownerID, subjectID); err != nil {
		return nil, err
	}

	profile, err := u.subjectProfileQuerySvc.GetSubjectProfileByID(ctx, profileUseCase.BuildSubjectProfileID(ownerID, subjectID))
	if err != nil {
		return nil, err
	}

	return profileUseCase.ToSubjectProfileResponse(profile, subjectID), nil
}

func (u *profileQueryUseCase) GetTagCatalog(ctx context.Context) (*respObj.TagCatalogResp, error) {
	catalog, err := u.tagCatalogQuerySvc.GetActiveTagCatalog(ctx)
	if err != nil {
		return nil, err
	}

	groupedTags := make(map[string][]respObj.TagItemResp, len(catalog.Groups))
	for _, tag := range catalog.Tags {
		groupedTags[tag.GroupKey] = append(groupedTags[tag.GroupKey], respObj.TagItemResp{
			GroupKey: tag.GroupKey,
			TagKey:   tag.TagKey,
			Label:    tag.Label,
		})
	}

	response := &respObj.TagCatalogResp{
		Groups: make([]respObj.TagGroupResp, 0, len(catalog.Groups)),
	}

	for _, group := range catalog.Groups {
		tags := groupedTags[group.GroupKey]
		if tags == nil {
			tags = []respObj.TagItemResp{}
		}

		response.Groups = append(response.Groups, respObj.TagGroupResp{
			GroupKey:      group.GroupKey,
			Label:         group.Label,
			SelectionMode: string(group.SelectionMode),
			Tags:          tags,
		})
	}

	return response, nil
}
