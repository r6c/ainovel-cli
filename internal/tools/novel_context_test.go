package tools

import (
	"github.com/voocel/ainovel-cli/internal/store"
)

func newTestContextTool(st *store.Store, refs References, style string) *ContextTool {
	return NewContextTool(st, refs, style, NewStyleStatsIndex(st))
}
