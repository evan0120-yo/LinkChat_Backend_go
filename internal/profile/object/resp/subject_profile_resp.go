package resp

import "time"

type SelectedTagResp struct {
	GroupKey string `json:"groupKey"`
	TagKey   string `json:"tagKey"`
}

type SubjectProfileResp struct {
	SubjectID    string            `json:"subjectId"`
	NoteLines    []string          `json:"noteLines"`
	SelectedTags []SelectedTagResp `json:"selectedTags"`
	UpdatedAt    *time.Time        `json:"updatedAt,omitempty"`
}
