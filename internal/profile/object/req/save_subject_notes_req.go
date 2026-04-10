package req

type SaveSubjectNotesReq struct {
	SubjectID string   `json:"subjectId" binding:"required"`
	Lines     []string `json:"lines"`
}
