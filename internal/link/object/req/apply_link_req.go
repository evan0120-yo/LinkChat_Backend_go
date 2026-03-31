package req

type ApplyLinkReq struct {
	RequesterID string // 誰發起的 (從 Token 解析)
	TargetID    string // 想加誰
}
