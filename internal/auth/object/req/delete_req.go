package req

type DeleteReq struct {
	UserID string `json:"userId" binding:"required"`
}
