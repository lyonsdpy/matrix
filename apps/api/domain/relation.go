package domain

// GraphRelation 图基础关系
type GraphRelation struct {
	ID     string         `json:"id"`
	FromID string         `json:"from_id"`
	ToID   string         `json:"to_id"`
	Type   string         `json:"type"`
	Props  map[string]any `json:"props"`
}
