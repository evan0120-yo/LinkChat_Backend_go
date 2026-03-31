package req

type ListLinkReq struct {
	// filter: "all", "active", "received", "sent"
	// 如果前端沒傳，預設就是 "all"
	Filter string `json:"filter"`
}
