package validator

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
)

const (
	maxNoteLines    = 3
	maxNoteLineRune = 60
)

type ProfileValidator interface {
	NormalizeNoteLines(lines []string) ([]string, error)
	ValidateAndNormalizeSelectedTags(selected []reqObj.SelectedTagReq, catalog *model.TagCatalog) ([]model.SelectedTag, error)
}

type profileValidator struct{}

func NewProfileValidator() ProfileValidator {
	return &profileValidator{}
}

func (v *profileValidator) NormalizeNoteLines(lines []string) ([]string, error) {
	normalized := make([]string, 0, len(lines))

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		if utf8.RuneCountInString(trimmed) > maxNoteLineRune {
			return nil, fmt.Errorf("line %d exceeds %d characters", idx+1, maxNoteLineRune)
		}

		normalized = append(normalized, trimmed)
	}

	if len(normalized) > maxNoteLines {
		return nil, fmt.Errorf("note lines cannot exceed %d items", maxNoteLines)
	}

	return normalized, nil
}

func (v *profileValidator) ValidateAndNormalizeSelectedTags(selected []reqObj.SelectedTagReq, catalog *model.TagCatalog) ([]model.SelectedTag, error) {
	if catalog == nil {
		return nil, fmt.Errorf("tag catalog is required")
	}

	groupMap := make(map[string]*model.TagGroup, len(catalog.Groups))
	for _, group := range catalog.Groups {
		groupMap[group.GroupKey] = group
	}

	tagMap := make(map[string]*model.TagDefinition, len(catalog.Tags))
	for _, tag := range catalog.Tags {
		tagMap[buildCompositeKey(tag.GroupKey, tag.TagKey)] = tag
	}

	normalized := make([]model.SelectedTag, 0, len(selected))
	deduped := make(map[string]struct{}, len(selected))
	groupSelections := make(map[string]int)

	for _, item := range selected {
		groupKey := normalizeKey(item.GroupKey)
		tagKey := normalizeKey(item.TagKey)

		if groupKey == "" || tagKey == "" {
			return nil, fmt.Errorf("groupKey and tagKey are required")
		}

		group, exists := groupMap[groupKey]
		if !exists {
			return nil, fmt.Errorf("tag group not found: %s", groupKey)
		}

		compositeKey := buildCompositeKey(groupKey, tagKey)
		if _, exists := tagMap[compositeKey]; !exists {
			return nil, fmt.Errorf("tag not found or inactive: %s", compositeKey)
		}

		if _, exists := deduped[compositeKey]; exists {
			continue
		}

		if group.SelectionMode == model.TagSelectionModeSingle && groupSelections[groupKey] >= 1 {
			return nil, fmt.Errorf("tag group only allows single selection: %s", groupKey)
		}

		deduped[compositeKey] = struct{}{}
		groupSelections[groupKey]++

		normalized = append(normalized, model.SelectedTag{
			GroupKey: groupKey,
			TagKey:   tagKey,
		})
	}

	return normalized, nil
}

func normalizeKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func buildCompositeKey(groupKey, tagKey string) string {
	return groupKey + "__" + tagKey
}
