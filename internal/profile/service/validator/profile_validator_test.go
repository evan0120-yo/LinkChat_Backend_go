package validator

import (
	"testing"

	"github.com/evan0120-yo/linkchat-go/internal/profile/model"
	reqObj "github.com/evan0120-yo/linkchat-go/internal/profile/object/req"
)

func TestNormalizeNoteLines(t *testing.T) {
	validator := NewProfileValidator()

	lines, err := validator.NormalizeNoteLines([]string{
		"  first note  ",
		"",
		" second note ",
	})
	if err != nil {
		t.Fatalf("NormalizeNoteLines returned error: %v", err)
	}

	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if lines[0] != "first note" || lines[1] != "second note" {
		t.Fatalf("unexpected normalized lines: %#v", lines)
	}
}

func TestNormalizeNoteLinesRejectsTooManyItems(t *testing.T) {
	validator := NewProfileValidator()

	_, err := validator.NormalizeNoteLines([]string{"1", "2", "3", "4"})
	if err == nil {
		t.Fatal("expected error for too many note lines")
	}
}

func TestValidateAndNormalizeSelectedTags(t *testing.T) {
	validator := NewProfileValidator()
	catalog := &model.TagCatalog{
		Groups: []*model.TagGroup{
			{GroupKey: "role", SelectionMode: model.TagSelectionModeSingle},
			{GroupKey: "support_need", SelectionMode: model.TagSelectionModeMulti},
		},
		Tags: []*model.TagDefinition{
			{GroupKey: "role", TagKey: "student"},
			{GroupKey: "support_need", TagKey: "reassurance"},
		},
	}

	selected, err := validator.ValidateAndNormalizeSelectedTags([]reqObj.SelectedTagReq{
		{GroupKey: " role ", TagKey: " student "},
		{GroupKey: "role", TagKey: "student"},
		{GroupKey: "support_need", TagKey: "reassurance"},
	}, catalog)
	if err != nil {
		t.Fatalf("ValidateAndNormalizeSelectedTags returned error: %v", err)
	}

	if len(selected) != 2 {
		t.Fatalf("expected 2 unique selections, got %d", len(selected))
	}
	if selected[0].GroupKey != "role" || selected[0].TagKey != "student" {
		t.Fatalf("unexpected first selection: %#v", selected[0])
	}
}

func TestValidateAndNormalizeSelectedTagsRejectsSingleGroupConflict(t *testing.T) {
	validator := NewProfileValidator()
	catalog := &model.TagCatalog{
		Groups: []*model.TagGroup{
			{GroupKey: "role", SelectionMode: model.TagSelectionModeSingle},
		},
		Tags: []*model.TagDefinition{
			{GroupKey: "role", TagKey: "student"},
			{GroupKey: "role", TagKey: "coworker"},
		},
	}

	_, err := validator.ValidateAndNormalizeSelectedTags([]reqObj.SelectedTagReq{
		{GroupKey: "role", TagKey: "student"},
		{GroupKey: "role", TagKey: "coworker"},
	}, catalog)
	if err == nil {
		t.Fatal("expected error for single-select group conflict")
	}
}
