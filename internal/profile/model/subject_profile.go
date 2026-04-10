package model

import "time"

type SelectedTag struct {
	GroupKey string `firestore:"group_key" json:"groupKey"`
	TagKey   string `firestore:"tag_key" json:"tagKey"`
}

type SubjectProfile struct {
	ID           string        `firestore:"id"`
	OwnerID      string        `firestore:"owner_id"`
	SubjectID    string        `firestore:"subject_id"`
	NoteLines    []string      `firestore:"note_lines"`
	SelectedTags []SelectedTag `firestore:"selected_tags"`
	UpdatedAt    time.Time     `firestore:"updated_at"`
}
