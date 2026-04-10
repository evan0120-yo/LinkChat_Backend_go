package req

type SelectedTagReq struct {
	GroupKey string `json:"groupKey" binding:"required"`
	TagKey   string `json:"tagKey" binding:"required"`
}

type SaveSubjectTagsReq struct {
	SubjectID string           `json:"subjectId" binding:"required"`
	Selected  []SelectedTagReq `json:"selected"`
}
