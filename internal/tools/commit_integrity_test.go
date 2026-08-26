package tools

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/voocel/ainovel-cli/internal/domain"
	"github.com/voocel/ainovel-cli/internal/errs"
	"github.com/voocel/ainovel-cli/internal/rules"
	"github.com/voocel/ainovel-cli/internal/store"
)

func TestCommitChapterRejectsTamperedSealedPayloadBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	originalPayload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "原始标题", "summary": "原始摘要",
		"characters": []string{"林墨"}, "key_events": []string{"原始事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tamperedPayload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "篡改标题", "summary": "篡改摘要",
		"characters": []string{"林墨"}, "key_events": []string{"篡改事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章冻结正文，长度足以证明恢复不应接受被单独替换的结构化载荷。"
	pendingFile, err := json.MarshalIndent(map[string]any{
		"chapter": 1, "stage": domain.CommitStageStarted,
		"payload": json.RawMessage(tamperedPayload), "draft_content": draft,
		"seal_version":   1,
		"payload_digest": fmt.Sprintf("%x", sha256.Sum256(originalPayload)),
		"draft_digest":   fmt.Sprintf("%x", sha256.Sum256([]byte(draft))),
		"intent_digest":  digestPendingIntent(domain.PendingCommit{Chapter: 1}),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), "meta", "pending_commit.json"), pendingFile, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("tampered sealed payload must be rejected")
	}
	if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("tampered payload error category = %v, want ErrPendingCommitIntegrity", err)
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("tampered recovery must not create chapter record: record=%+v err=%v", record, loadErr)
	}
	progress, loadErr := s.Progress.Load()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if len(progress.CompletedChapters) != 0 {
		t.Fatalf("tampered recovery advanced progress: %+v", progress.CompletedChapters)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("tampered pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterRejectsTamperedSealedDraftBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "原始标题", "summary": "原始摘要",
		"characters": []string{"林墨"}, "key_events": []string{"原始事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	originalDraft := "第一章原始冻结正文，恢复只能使用这一份经过密封的正文快照。"
	tamperedDraft := "第一章篡改正文，虽然仍是合法文本，但摘要与冻结时不一致。"
	pendingFile, err := json.MarshalIndent(map[string]any{
		"chapter": 1, "stage": domain.CommitStageStarted,
		"payload": json.RawMessage(payload), "draft_content": tamperedDraft,
		"seal_version":   1,
		"payload_digest": fmt.Sprintf("%x", sha256.Sum256(payload)),
		"draft_digest":   fmt.Sprintf("%x", sha256.Sum256([]byte(originalDraft))),
		"intent_digest":  digestPendingIntent(domain.PendingCommit{Chapter: 1}),
	}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(s.Dir(), "meta", "pending_commit.json"), pendingFile, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("tampered sealed draft must be rejected")
	}
	if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("tampered draft error category = %v, want ErrPendingCommitIntegrity", err)
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("tampered draft recovery must not create chapter record: record=%+v err=%v", record, loadErr)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("tampered pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterRejectsOriginOnUnsealedLegacyPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Origin: domain.ChapterOriginImported,
		Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}

	_, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("unsealed legacy pending cannot acquire imported provenance, got %v", err)
	}
	got, loadErr := s.Signals.LoadPendingCommit()
	if loadErr != nil || got == nil || got.SealVersion != 0 {
		t.Fatalf("invalid legacy pending must remain unsealed: pending=%+v err=%v", got, loadErr)
	}
}

func TestCommitChapterRejectsOriginTamperOnLegacyV1Seal(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":[],"key_events":["事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending := domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文"}
	payloadDigest, err := digestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	pending.SealVersion = 1
	pending.PayloadDigest = payloadDigest
	pending.DraftDigest = fmt.Sprintf("%x", sha256.Sum256([]byte(pending.DraftContent)))
	pending.IntentDigest = digestPendingIntent(pending)
	pending.Origin = domain.ChapterOriginImported // v1 历史格式未密封 origin，必须拒绝而非升级权限。
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !errors.Is(err, errs.ErrPendingCommitIntegrity) {
		t.Fatalf("v1 origin tamper must fail integrity validation, got %v", err)
	}
	if got, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || got == nil {
		t.Fatalf("tampered pending must remain: pending=%+v err=%v", got, loadErr)
	}
}

func TestCommitChapterRecoversImportedPendingWithFrozenProvenance(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.StartChapter(1); err != nil {
		t.Fatal(err)
	}
	if err := s.UserRules.Save(&rules.Snapshot{Version: rules.SnapshotVersion, Status: rules.StatusReady,
		Structured: rules.Structured{ChapterTargetChars: 10}}); err != nil {
		t.Fatal(err)
	}
	content := "# 第一章\n\n导入的**原始强调**必须保留。\n"
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"导入摘要","characters":[],"key_events":["导入事件"],"timeline_events":[],"foreshadow_updates":[],"relationship_changes":[],"state_changes":[],"knowledge_updates":[],"cast_intros":[]}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Origin: domain.ChapterOriginImported,
		Payload: payload, DraftContent: content,
	})
	if err != nil {
		t.Fatal(err)
	}
	if pending.SealVersion != 2 {
		t.Fatalf("imported pending seal version=%d want=2", pending.SealVersion)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("recover imported pending through ordinary resume entry: %v", err)
	}
	record, err := s.ChapterRecords.Load(1)
	if err != nil {
		t.Fatal(err)
	}
	if record == nil || record.Origin != domain.ChapterOriginImported || record.Content != content {
		t.Fatalf("imported provenance/content lost on recovery: %+v", record)
	}
	if got, err := s.Signals.LoadPendingCommit(); err != nil || got != nil {
		t.Fatalf("imported pending not cleared: pending=%+v err=%v", got, err)
	}
}

func TestCommitChapterRejectsMalformedPendingSealBeforeRecoverySideEffects(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章合法冻结正文。"
	payloadDigest := fmt.Sprintf("%x", sha256.Sum256(payload))
	draftDigest := fmt.Sprintf("%x", sha256.Sum256([]byte(draft)))
	tests := []struct {
		name    string
		pending domain.PendingCommit
	}{
		{
			name: "half sealed legacy format",
			pending: domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
				PayloadDigest: payloadDigest},
		},
		{
			name: "unknown seal version",
			pending: domain.PendingCommit{Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
				SealVersion: 2, PayloadDigest: payloadDigest, DraftDigest: draftDigest},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			if err := s.Signals.SavePendingCommit(tt.pending); err != nil {
				t.Fatal(err)
			}
			_, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil {
				t.Fatal("malformed pending seal must be rejected")
			}
			if !errors.Is(err, errs.ErrPendingCommitIntegrity) {
				t.Fatalf("malformed seal error category = %v, want ErrPendingCommitIntegrity", err)
			}
			if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
				t.Fatalf("malformed seal must fail before chapter record: record=%+v err=%v", record, loadErr)
			}
			if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
				t.Fatalf("malformed pending must be preserved: pending=%+v err=%v", pending, loadErr)
			}
		})
	}
}

func TestCommitChapterRecoversSealedStateAppliedPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	draft := "第一章已经完成正文与状态落盘，恢复只应从进度阶段继续。"
	if err := s.Drafts.SaveFinalChapter(1, draft); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, draft, domain.ChapterFacts{
		Title: "第一章", Summary: "摘要", Characters: []string{"林墨"}, KeyEvents: []string{"事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatal(err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{
		Chapter: 1, Title: "第一章", Summary: "摘要", Characters: []string{"林墨"}, KeyEvents: []string{"事件"},
	}); err != nil {
		t.Fatal(err)
	}
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStateApplied, Payload: payload, DraftContent: draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err != nil {
		t.Fatalf("recover state_applied: %v", err)
	}
	progress, err := s.Progress.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Contains(progress.CompletedChapters, 1) {
		t.Fatalf("state_applied recovery did not mark progress: %+v", progress)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("state_applied recovery did not clear pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRecoversSealedSignalSavedPending(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.MarkChapterComplete(1, 100, "", ""); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	output := json.RawMessage(`{"chapter":1,"committed":true,"sealed":true}`)
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageSignalSaved, Payload: payload, DraftContent: "第一章冻结正文", Output: output,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	got, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatalf("recover signal_saved: %v", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, got); err != nil {
		t.Fatal(err)
	}
	if compact.String() != string(output) {
		t.Fatalf("signal_saved output=%s want=%s", got, output)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("signal_saved recovery did not clear pending: pending=%+v err=%v", pending, err)
	}
}

func TestCommitChapterRejectsTamperedSealAtEveryRecoveryStage(t *testing.T) {
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	originalDraft := "原始冻结正文"
	for _, stage := range []domain.CommitStage{
		domain.CommitStageStarted,
		domain.CommitStageStateApplied,
		domain.CommitStageProgressMarked,
		domain.CommitStageSignalSaved,
	} {
		t.Run(string(stage), func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			pending, err := sealPendingCommit(domain.PendingCommit{
				Chapter: 1, Stage: stage, Payload: payload, DraftContent: originalDraft,
			})
			if err != nil {
				t.Fatal(err)
			}
			pending.DraftContent = "篡改冻结正文"
			if err := s.Signals.SavePendingCommit(pending); err != nil {
				t.Fatal(err)
			}
			_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil || !strings.Contains(err.Error(), "draft 摘要不匹配") {
				t.Fatalf("stage %s must reject tampered seal, got %v", stage, err)
			}
			got, loadErr := s.Signals.LoadPendingCommit()
			if loadErr != nil || got == nil || got.Stage != stage {
				t.Fatalf("stage %s pending not preserved: pending=%+v err=%v", stage, got, loadErr)
			}
		})
	}
}

func TestCommitChapterRejectsIncoherentPendingMetadataBeforeRecoverySideEffects(t *testing.T) {
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "摘要",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
	})
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		draft       string
		rewrite     bool
		rewriteMode string
	}{
		{name: "empty draft"},
		{name: "ordinary commit with rewrite mode", draft: "冻结正文", rewriteMode: "rewrite"},
		{name: "rewrite with unknown mode", draft: "冻结正文", rewrite: true, rewriteMode: "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := store.NewStore(t.TempDir())
			if err := s.Init(); err != nil {
				t.Fatal(err)
			}
			if err := s.Progress.Init(1); err != nil {
				t.Fatal(err)
			}
			pending, err := sealPendingCommit(domain.PendingCommit{
				Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: tt.draft,
				Rewrite: tt.rewrite, RewriteMode: tt.rewriteMode,
			})
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Signals.SavePendingCommit(pending); err != nil {
				t.Fatal(err)
			}
			_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
			if err == nil {
				t.Fatal("incoherent pending metadata must be rejected")
			}
			if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
				t.Fatalf("incoherent pending must fail before chapter record: record=%+v err=%v", record, loadErr)
			}
		})
	}
}

func TestCommitChapterRejectsSealedPayloadWithInvalidFactsBeforeRecoverySideEffects(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "非法知识字段",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
		"knowledge_updates": []map[string]any{{"id": "k", "action": "establish"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章冻结正文，结构化事实虽然摘要匹配，但字段矩阵本身非法。"
	pending, err := sealPendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Signals.SavePendingCommit(pending); err != nil {
		t.Fatal(err)
	}

	_, err = newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil {
		t.Fatal("sealed payload with invalid chapter facts must be rejected")
	}
	if record, loadErr := s.ChapterRecords.Load(1); loadErr != nil || record != nil {
		t.Fatalf("invalid frozen payload must fail before chapter record: record=%+v err=%v", record, loadErr)
	}
	if pending, loadErr := s.Signals.LoadPendingCommit(); loadErr != nil || pending == nil {
		t.Fatalf("invalid frozen pending must be preserved: pending=%+v err=%v", pending, loadErr)
	}
}

func TestCommitChapterSealsLegacyPendingBeforeReplayingState(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(map[string]any{
		"chapter": 1, "title": "第一章", "summary": "推进未知伏笔",
		"characters": []string{"林墨"}, "key_events": []string{"事件"},
		"foreshadow_updates": []map[string]any{{"id": "missing", "action": "advance"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	draft := "第一章旧版冻结正文，恢复时会在伏笔状态应用阶段失败。"
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: draft,
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("state replay should fail on unknown foreshadow")
	}
	pending, err := s.Signals.LoadPendingCommit()
	if err != nil || pending == nil {
		t.Fatalf("legacy pending must remain after failed replay: pending=%+v err=%v", pending, err)
	}
	if pending.SealVersion != 2 || pending.Origin != domain.ChapterOriginGenerated || len(pending.PayloadDigest) != 64 || len(pending.DraftDigest) != 64 {
		t.Fatalf("legacy pending was not sealed before replay: %+v", pending)
	}
	wantPayload, err := digestPayload(payload)
	if err != nil {
		t.Fatal(err)
	}
	wantDraft := fmt.Sprintf("%x", sha256.Sum256([]byte(draft)))
	if pending.PayloadDigest != wantPayload || pending.DraftDigest != wantDraft {
		t.Fatalf("legacy pending seal does not match frozen inputs: %+v", pending)
	}
}

func TestCommitChapterDoesNotSealInvalidLegacyPayload(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","key_events":[]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}

	if _, err := newTestCommitChapterTool(s).Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("invalid legacy payload must be rejected")
	}
	pending, err := s.Signals.LoadPendingCommit()
	if err != nil || pending == nil {
		t.Fatalf("invalid legacy pending must remain: pending=%+v err=%v", pending, err)
	}
	if pending.SealVersion != 0 || pending.PayloadDigest != "" || pending.DraftDigest != "" {
		t.Fatalf("invalid legacy payload must not be sealed: %+v", pending)
	}
	if record, err := s.ChapterRecords.Load(1); err != nil || record != nil {
		t.Fatalf("invalid legacy payload created chapter record: record=%+v err=%v", record, err)
	}
}

func TestCommitChapterRejectsLegacyPayloadModifiedAfterAutomaticSeal(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatal(err)
	}
	if err := s.Progress.Init(1); err != nil {
		t.Fatal(err)
	}
	payload := json.RawMessage(`{"chapter":1,"title":"第一章","summary":"摘要","characters":["林墨"],"key_events":["事件"],"foreshadow_updates":[{"id":"missing","action":"advance"}]}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1, Stage: domain.CommitStageStarted, Payload: payload, DraftContent: "冻结正文",
	}); err != nil {
		t.Fatal(err)
	}
	tool := newTestCommitChapterTool(s)
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`)); err == nil {
		t.Fatal("first replay must fail after sealing at state application")
	}
	sealed, err := s.Signals.LoadPendingCommit()
	if err != nil || sealed == nil || sealed.SealVersion != 2 || sealed.Origin != domain.ChapterOriginGenerated {
		t.Fatalf("legacy pending was not sealed: pending=%+v err=%v", sealed, err)
	}
	sealed.Payload = json.RawMessage(`{"chapter":1,"title":"篡改标题","summary":"摘要","characters":["林墨"],"key_events":["事件"]}`)
	if err := s.Signals.SavePendingCommit(*sealed); err != nil {
		t.Fatal(err)
	}

	_, err = tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err == nil || !strings.Contains(err.Error(), "摘要不匹配") {
		t.Fatalf("modified auto-sealed pending must fail integrity check, got %v", err)
	}
}

func TestCommitChapterReplayAfterPartialCommitDoesNotDuplicateWorldState(t *testing.T) {
	dir := t.TempDir()
	s := store.NewStore(dir)
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(10); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "第二章正文，林墨遇到黑影并突破。"); err != nil {
		t.Fatalf("SaveDraft: %v", err)
	}

	timeline := []domain.TimelineEvent{{
		Chapter:    2,
		Time:       "清晨",
		Event:      "林墨遇到黑影",
		Characters: []string{"林墨"},
	}}
	stateChanges := []domain.StateChange{{
		Chapter:  2,
		Entity:   "林墨",
		Field:    "realm",
		OldValue: "凡人",
		NewValue: "练气期",
	}}
	if err := s.World.UpdateForeshadow(1, []domain.ForeshadowUpdate{{
		ID: "f1", Action: "plant", Description: "黑影身份",
	}}); err != nil {
		t.Fatalf("plant foreshadow seed: %v", err)
	}
	foreshadow := []domain.ForeshadowUpdate{{
		ID:     "f1",
		Action: "reinforce",
	}}

	// 模拟 commit_chapter 已写入世界状态，但尚未 MarkChapterComplete 时进程崩溃。
	if err := s.World.AppendTimelineEvents(timeline); err != nil {
		t.Fatalf("AppendTimelineEvents seed: %v", err)
	}
	if err := s.World.AppendStateChanges(stateChanges); err != nil {
		t.Fatalf("AppendStateChanges seed: %v", err)
	}
	if err := s.World.UpdateForeshadow(2, foreshadow); err != nil {
		t.Fatalf("UpdateForeshadow seed: %v", err)
	}
	persistedArgs, _ := json.Marshal(map[string]any{
		"chapter":            2,
		"title":              "第二章",
		"summary":            "林墨遇到黑影并突破",
		"characters":         []string{"林墨"},
		"key_events":         []string{"遇到黑影", "突破"},
		"timeline_events":    timeline,
		"state_changes":      stateChanges,
		"foreshadow_updates": foreshadow,
	})
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter:      2,
		Stage:        domain.CommitStageStarted,
		Summary:      "半提交摘要",
		Payload:      persistedArgs,
		DraftContent: "第二章正文，林墨遇到黑影并突破。",
	}); err != nil {
		t.Fatalf("SavePendingCommit: %v", err)
	}
	if err := s.Drafts.SaveDraft(2, "重启后被新 Worker 覆盖、绝不能混入旧提交的正文。"); err != nil {
		t.Fatalf("overwrite draft: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	// 模拟重启后的 Writer 重新生成了不同参数；恢复必须忽略它，使用 persistedArgs。
	args, _ := json.Marshal(map[string]any{
		"chapter":         2,
		"title":           "错误标题",
		"summary":         "错误的新摘要",
		"characters":      []string{"林墨"},
		"key_events":      []string{"错误事件"},
		"timeline_events": []domain.TimelineEvent{{Time: "夜晚", Event: "不应写入的新事件"}},
	})
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("Execute replay: %v", err)
	}

	events, _ := s.World.LoadTimeline()
	if len(events) != 1 {
		t.Fatalf("timeline duplicated after replay, got %d: %+v", len(events), events)
	}
	changes, _ := s.World.LoadStateChanges()
	if len(changes) != 1 {
		t.Fatalf("state changes duplicated after replay, got %d: %+v", len(changes), changes)
	}
	ledger, _ := s.World.LoadForeshadowLedger()
	if len(ledger) != 1 || ledger[0].Status != "reinforced" || ledger[0].LastAdvancedAt != 2 {
		t.Fatalf("foreshadow reinforce changed after replay: %+v", ledger)
	}
	pending, _ := s.Signals.LoadPendingCommit()
	if pending != nil {
		t.Fatalf("pending commit should be cleared, got %+v", pending)
	}
	if cp := s.Checkpoints.LatestByStep(domain.ChapterScope(2), "commit"); cp == nil {
		t.Fatal("commit checkpoint should be written")
	}
	final, err := s.Drafts.LoadChapterText(2)
	if err != nil {
		t.Fatalf("LoadChapterText: %v", err)
	}
	if final != "第二章正文，林墨遇到黑影并突破。" {
		t.Fatalf("recovery used overwritten draft: %q", final)
	}
}

func TestCommitChapterRecoversProgressMarkedWindowWithExactOutput(t *testing.T) {
	s := store.NewStore(t.TempDir())
	if err := s.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if err := s.Progress.Init(2); err != nil {
		t.Fatalf("InitProgress: %v", err)
	}
	if err := s.Progress.UpdatePhase(domain.PhaseWriting); err != nil {
		t.Fatalf("UpdatePhase: %v", err)
	}
	if err := s.Progress.StartChapter(1); err != nil {
		t.Fatalf("StartChapter: %v", err)
	}
	if err := s.Drafts.SaveFinalChapter(1, "第一章终稿"); err != nil {
		t.Fatalf("SaveFinalChapter: %v", err)
	}
	if err := s.Summaries.SaveSummary(domain.ChapterSummary{Chapter: 1, Title: "第一章", Summary: "摘要"}); err != nil {
		t.Fatalf("SaveSummary: %v", err)
	}
	if _, err := s.ChapterRecords.Accept(1, domain.ChapterOriginGenerated, "第一章终稿", domain.ChapterFacts{
		Title: "第一章", Summary: "摘要", KeyEvents: []string{"事件"},
	}, domain.StyleDelta{}); err != nil {
		t.Fatalf("SaveChapterRecord: %v", err)
	}
	if err := s.Progress.MarkChapterComplete(1, 100, "mystery", "quest"); err != nil {
		t.Fatalf("MarkChapterComplete: %v", err)
	}

	want := json.RawMessage(`{"chapter":1,"committed":true,"recovered":"exact"}`)
	if err := s.Signals.SavePendingCommit(domain.PendingCommit{
		Chapter: 1,
		Stage:   domain.CommitStageProgressMarked,
		Output:  want,
	}); err != nil {
		t.Fatalf("SavePendingCommit: %v", err)
	}

	tool := newTestCommitChapterTool(s)
	got, err := tool.Execute(context.Background(), json.RawMessage(`{"chapter":1}`))
	if err != nil {
		t.Fatalf("Execute recovery: %v", err)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, got); err != nil {
		t.Fatalf("compact recovered output: %v", err)
	}
	if compact.String() != string(want) {
		t.Fatalf("recovered output = %s, want exact document %s", got, want)
	}
	if pending, err := s.Signals.LoadPendingCommit(); err != nil || pending != nil {
		t.Fatalf("pending commit should be cleared, pending=%+v err=%v", pending, err)
	}
	if cp := s.Checkpoints.LatestByStep(domain.ChapterScope(1), "commit"); cp == nil {
		t.Fatal("commit checkpoint should be repaired")
	}
	p, err := s.Progress.Load()
	if err != nil {
		t.Fatalf("LoadProgress: %v", err)
	}
	if p.InProgressChapter != 0 {
		t.Fatalf("in-progress chapter should be cleared, got %d", p.InProgressChapter)
	}
}
