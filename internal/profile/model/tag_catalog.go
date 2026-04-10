package model

type TagSelectionMode string

const (
	TagSelectionModeSingle TagSelectionMode = "single"
	TagSelectionModeMulti  TagSelectionMode = "multi"
)

type TagGroup struct {
	GroupKey      string           `firestore:"group_key"`
	Label         string           `firestore:"label"`
	SelectionMode TagSelectionMode `firestore:"selection_mode"`
	Active        bool             `firestore:"active"`
	OrderNo       int              `firestore:"order_no"`
}

type TagDefinition struct {
	ID       string `firestore:"id"`
	GroupKey string `firestore:"group_key"`
	TagKey   string `firestore:"tag_key"`
	Label    string `firestore:"label"`
	Active   bool   `firestore:"active"`
	OrderNo  int    `firestore:"order_no"`
}

type TagCatalog struct {
	Groups []*TagGroup
	Tags   []*TagDefinition
}
