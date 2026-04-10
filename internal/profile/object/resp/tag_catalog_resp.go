package resp

type TagCatalogResp struct {
	Groups []TagGroupResp `json:"groups"`
}

type TagGroupResp struct {
	GroupKey      string        `json:"groupKey"`
	Label         string        `json:"label"`
	SelectionMode string        `json:"selectionMode"`
	Tags          []TagItemResp `json:"tags"`
}

type TagItemResp struct {
	GroupKey string `json:"groupKey"`
	TagKey   string `json:"tagKey"`
	Label    string `json:"label"`
}
