package domain

// KnowledgeHolder 记录角色首次获知某项作者真相的章节。
type KnowledgeHolder struct {
	Character string `json:"character"`
	LearnedAt int    `json:"learned_at"`
}

// KnowledgeEntry 是作者真相及其角色知情范围的当前投影。
type KnowledgeEntry struct {
	ID               string            `json:"id"`
	Truth            string            `json:"truth"`
	EstablishedAt    int               `json:"established_at"`
	KnownBy          []KnowledgeHolder `json:"known_by"`
	ReaderRevealedAt int               `json:"reader_revealed_at,omitempty"`
}

// KnowledgeUpdate 是章节对知识状态的增量操作。
type KnowledgeUpdate struct {
	ID        string `json:"id"`
	Action    string `json:"action"` // establish / learn / reveal_to_reader
	Truth     string `json:"truth,omitempty"`
	Character string `json:"character,omitempty"`
}

// StateChange 角色/实体状态变化记录。
type StateChange struct {
	Chapter  int    `json:"chapter"`
	Entity   string `json:"entity"`              // 角色名或实体名
	Field    string `json:"field"`               // 变化属性：realm/location/status/power/relation 等
	OldValue string `json:"old_value,omitempty"` // 变化前（首次出现可空）
	NewValue string `json:"new_value"`           // 变化后
	Reason   string `json:"reason,omitempty"`    // 变化原因
}
