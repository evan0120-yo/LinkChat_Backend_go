package seeder

import (
	"context"
	"fmt"

	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"github.com/evan0120-yo/linkchat-go/internal/profile/repository"
)

type ProfileSeeder struct {
	tagCatalogRepository repository.TagCatalogRepository
}

func NewProfileSeeder(tagCatalogRepository repository.TagCatalogRepository) *ProfileSeeder {
	return &ProfileSeeder{
		tagCatalogRepository: tagCatalogRepository,
	}
}

func (s *ProfileSeeder) Seed(ctx context.Context) error {
	fmt.Println("[ProfileSeeder] 正在植入 persona tag catalog...")

	groups := []*model.TagGroup{
		{GroupKey: "role", Label: "角色", SelectionMode: model.TagSelectionModeSingle, Active: true, OrderNo: 10},
		{GroupKey: "communication_style", Label: "互動風格", SelectionMode: model.TagSelectionModeMulti, Active: true, OrderNo: 20},
		{GroupKey: "support_need", Label: "支持需求", SelectionMode: model.TagSelectionModeMulti, Active: true, OrderNo: 30},
	}

	tags := []*model.TagDefinition{
		{ID: "role__student", GroupKey: "role", TagKey: "student", Label: "學生", Active: true, OrderNo: 10},
		{ID: "role__coworker", GroupKey: "role", TagKey: "coworker", Label: "同事", Active: true, OrderNo: 20},
		{ID: "role__family", GroupKey: "role", TagKey: "family", Label: "家人", Active: true, OrderNo: 30},
		{ID: "communication_style__slow_warmup", GroupKey: "communication_style", TagKey: "slow_warmup", Label: "慢熟", Active: true, OrderNo: 10},
		{ID: "communication_style__direct", GroupKey: "communication_style", TagKey: "direct", Label: "直接", Active: true, OrderNo: 20},
		{ID: "communication_style__step_by_step", GroupKey: "communication_style", TagKey: "step_by_step", Label: "步驟式", Active: true, OrderNo: 30},
		{ID: "support_need__reassurance", GroupKey: "support_need", TagKey: "reassurance", Label: "需要安撫", Active: true, OrderNo: 10},
		{ID: "support_need__space_first", GroupKey: "support_need", TagKey: "space_first", Label: "先留空間", Active: true, OrderNo: 20},
	}

	for _, group := range groups {
		if err := s.tagCatalogRepository.SaveGroup(ctx, group); err != nil {
			return err
		}
	}

	for _, tag := range tags {
		if err := s.tagCatalogRepository.SaveTag(ctx, tag); err != nil {
			return err
		}
	}

	return nil
}
