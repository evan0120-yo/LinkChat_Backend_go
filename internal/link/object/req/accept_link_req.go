package req

type AcceptLinkReq struct {
	OperatorID string // 誰執行的 (從 Token 解析，確保安全)
	LinkID     string // 連結 ID
}
