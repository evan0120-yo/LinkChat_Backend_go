package repository

import (
	"context"
	"fmt"
	"sort"

	"cloud.google.com/go/firestore"
	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	"google.golang.org/api/iterator"
)

type TagCatalogRepository interface {
	SaveGroup(ctx context.Context, group *model.TagGroup) error
	SaveTag(ctx context.Context, tag *model.TagDefinition) error
	ListActiveGroups(ctx context.Context) ([]*model.TagGroup, error)
	ListActiveTags(ctx context.Context) ([]*model.TagDefinition, error)
}

type firestoreTagCatalogRepository struct {
	client *firestore.Client
}

func NewTagCatalogRepository(client *firestore.Client) TagCatalogRepository {
	return &firestoreTagCatalogRepository{client: client}
}

func (r *firestoreTagCatalogRepository) SaveGroup(ctx context.Context, group *model.TagGroup) error {
	_, err := r.client.Collection("profile_tag_groups").Doc(group.GroupKey).Set(ctx, group)
	if err != nil {
		return fmt.Errorf("failed to save tag group: %w", err)
	}
	return nil
}

func (r *firestoreTagCatalogRepository) SaveTag(ctx context.Context, tag *model.TagDefinition) error {
	_, err := r.client.Collection("profile_tags").Doc(tag.ID).Set(ctx, tag)
	if err != nil {
		return fmt.Errorf("failed to save tag definition: %w", err)
	}
	return nil
}

func (r *firestoreTagCatalogRepository) ListActiveGroups(ctx context.Context) ([]*model.TagGroup, error) {
	iter := r.client.Collection("profile_tag_groups").Where("active", "==", true).Documents(ctx)
	defer iter.Stop()

	var groups []*model.TagGroup
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tag groups: %w", err)
		}

		var group model.TagGroup
		if err := doc.DataTo(&group); err != nil {
			continue
		}
		groups = append(groups, &group)
	}

	sort.Slice(groups, func(i, j int) bool {
		if groups[i].OrderNo != groups[j].OrderNo {
			return groups[i].OrderNo < groups[j].OrderNo
		}
		return groups[i].GroupKey < groups[j].GroupKey
	})

	return groups, nil
}

func (r *firestoreTagCatalogRepository) ListActiveTags(ctx context.Context) ([]*model.TagDefinition, error) {
	iter := r.client.Collection("profile_tags").Where("active", "==", true).Documents(ctx)
	defer iter.Stop()

	var tags []*model.TagDefinition
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to iterate tags: %w", err)
		}

		var tag model.TagDefinition
		if err := doc.DataTo(&tag); err != nil {
			continue
		}
		tags = append(tags, &tag)
	}

	sort.Slice(tags, func(i, j int) bool {
		if tags[i].GroupKey != tags[j].GroupKey {
			return tags[i].GroupKey < tags[j].GroupKey
		}
		if tags[i].OrderNo != tags[j].OrderNo {
			return tags[i].OrderNo < tags[j].OrderNo
		}
		return tags[i].TagKey < tags[j].TagKey
	})

	return tags, nil
}
