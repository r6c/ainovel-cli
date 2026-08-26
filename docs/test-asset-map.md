# 测试资产归属表

> 阶段 180 产物。只记录当前测试函数与 seam 的归属，不代表已经移动文件。候选 4 的后续移动必须保持测试名称、数量、公共接口和 `-run` 筛选兼容。
>
> 盘点基线：`230ce1f 文档：收口发布候选计划状态`。本文件创建时没有修改任何 Go 文件。

## 1. Commit 测试

当前文件：`internal/tools/commit_chapter_test.go`（约 3645 行）。共享构造入口：`newTestCommitChapterTool`、`saveTestChapterRecord`。

| 目标 seam | 预期文件 | 测试范围 |
|---|---|---|
| 纯字段/篇幅/格式与 Schema | `commit_payload_test.go` | `TestChapterTargetMaxUsesBoundedOverflowSafeCalculation`; `TestCommitChapterRejectsPersistedChapterTargetAboveProductLimit`; `TestCommitChapterRejectsChapterOverTargetBeforePendingCommit`; `TestCommitChapterDoesNotBlockChapterBelowTarget`; `TestCommitChapterRewriteRejectsChapterOverTargetBeforePendingCommit`; `TestCommitChapterRejectsMarkdownResidueBeforePendingCommit`; `TestCommitChapterRewriteRejectsMarkdownResidueBeforePendingCommit`; `TestCommitChapterPersistsDuplicateParagraphViolationWithoutBlocking`; `TestCommitChapterSchemaIncludesKnowledgeUpdates`; `TestCommitChapterSchemaDescribesFeedbackAsObject`; `TestCommitChapterRejectsInvalidNestedFields`（已移动到 `commit_payload_test.go`） |
| Knowledge 前置与正常应用 | `commit_knowledge_test.go` | `TestCommitChapterRejectsConflictingFutureKnowledgeBeforePending`; `TestCommitChapterRejectsConflictingKnowledgeBeforePending`; `TestCommitChapterRejectsLearningUnknownKnowledgeBeforePending`; `TestCommitChapterRejectsBelievingUnknownKnowledgeBeforePending`; `TestCommitChapterRejectsTrueBeliefBeforePending`; `TestCommitChapterRejectsBeliefAfterCharacterKnowsTruthBeforePending`; `TestCommitChapterRejectsBeliefAfterLearningInSamePayloadBeforePending`; `TestCommitChapterRejectsChangingActiveBeliefBeforePending`; `TestCommitChapterRejectsRevealingUnknownKnowledgeBeforePending`; `TestCommitChapterRevealsKnowledgeToReader`; `TestCommitChapterRecordsCharacterFalseBelief`; `TestCommitChapterEstablishesBeliefAndLearningInSamePayload`; `TestCommitChapterRecordsCharacterLearningKnowledge`; `TestCommitChapterEstablishesAndRevealsKnowledgeInSamePayload`; `TestCommitChapterEstablishesAndLearnsKnowledgeInSamePayload`; `TestCommitChapterEstablishesKnowledge`; `TestCommitChapterReplayKeepsFirstReaderRevealChapter`; `TestCommitChapterReplayDoesNotDuplicateKnowledgeState`; `TestCommitChapterRewriteRejectsRemovingKnowledgeRequiredByLaterChapterBeforePending`; `TestCommitChapterRewriteRebuildsKnowledgeState` |
| Foreshadow 前置与正常应用 | `commit_foreshadow_test.go` | `TestCommitChapterRejectsUnknownForeshadowReferenceBeforePending`; `TestCommitChapterReinforcesForeshadow`; `TestCommitChapterRejectsAdvancingResolvedForeshadowBeforePending`; `TestCommitChapterRejectsReinforcingResolvedForeshadowBeforePending`; `TestCommitChapterRejectsActionsAfterResolveInSamePayloadBeforePending`; `TestCommitChapterRewriteRebuildsForeshadowLifecycle`; `TestCommitChapterRewriteKeepsOwnForeshadowPlant`; `TestCommitChapterRewriteRejectsForwardForeshadowReference`; `TestCommitChapterReplayKeepsSameChapterAdvancedThenResolvedForeshadow` |
| Rewrite/provenance/返工权限 | `commit_rewrite_test.go` | `TestCommitChapterRejectsNonPendingRewrite`; `TestCommitChapterRejectsAutomaticRewriteOfImportedChapter`; `TestCommitChapterAllowsPendingRewrite`; `TestCommitChapterRefreshesSharedStyleStatsAfterRewrite`; `TestCommitChapterRejectsPolishWithoutDraftChange`; `TestCommitChapterAllowsTitleOnlyRewrite`; `TestCommitChapterLayeredRejectsOutOfRangeChapter`; `TestCommitChapterRejectsTamperedRewriteIntent`; `TestCommitChapterRewriteRecoveryUsesFrozenDraft` |
| PendingCommit 密封与多阶段恢复 | `commit_integrity_test.go` | `TestSealPendingCommitAddsStablePayloadAndDraftDigests`; `TestCommitChapterRejectsTamperedSealedPayloadBeforeRecoverySideEffects`; `TestCommitChapterRejectsTamperedSealedDraftBeforeRecoverySideEffects`; `TestCommitChapterRejectsOriginOnUnsealedLegacyPending`; `TestCommitChapterRejectsOriginTamperOnLegacyV1Seal`; `TestExecuteImportedCannotChangeFrozenGeneratedProvenance`; `TestCommitChapterRecoversImportedPendingWithFrozenProvenance`; `TestCommitChapterRejectsMalformedPendingSealBeforeRecoverySideEffects`; `TestCommitChapterRecoversSealedStateAppliedPending`; `TestCommitChapterRecoversSealedSignalSavedPending`; `TestCommitChapterRejectsTamperedSealAtEveryRecoveryStage`; `TestCommitChapterRejectsIncoherentPendingMetadataBeforeRecoverySideEffects`; `TestCommitChapterRejectsSealedPayloadWithInvalidFactsBeforeRecoverySideEffects`; `TestCommitChapterSealsLegacyPendingBeforeReplayingState`; `TestCommitChapterDoesNotSealInvalidLegacyPayload`; `TestCommitChapterRejectsLegacyPayloadModifiedAfterAutomaticSeal`; `TestCommitChapterReplayAfterPartialCommitDoesNotDuplicateWorldState`; `TestCommitChapterRecoversProgressMarkedWindowWithExactOutput` |
| Completion/层级推进 | `commit_rewrite_test.go` | `TestCommitChapterNonLayeredRecompletesAfterRework`; `TestCommitChapterLayeredReopenRecompletesDespiteOpenThread`; `TestCommitChapterLayeredAutoCompletesWhenDone`; `TestCommitChapterFinaleVolumeCompletesDespiteOpenThreads`; `TestCommitChapterFinaleSkeletonArcBlocksCompletion`; `TestCommitChapterLayeredNoAutoCompleteWithOpenThreads` |
| Saga 其他事实/目录更新 | `commit_side_effects_test.go` | `TestCommitChapterUpdatesCastLedger` |

说明：`TestCommitChapterRejectsUnknownForeshadowReferenceBeforePending` 仍按主要断言归入 Foreshadow；`TestCommitChapterSchema...` 属于 payload seam，不随 Knowledge 文件移动。Completion 测试与 Rewrite/层级推进测试共用 `commit_rewrite_test.go`；Cast Ledger 单独位于 `commit_side_effects_test.go`。跨进程恢复保留在 `commit_process_recovery_test.go`。若一个测试跨多个 seam，归属以主要失败断言/公共接口为准，并在迁移前保留原测试文件中的共享 helper。

## 2. Context 测试

当前文件：`internal/tools/novel_context_test.go`（约 1516 行）。共享构造入口：`newTestContextTool`。

| 目标 seam | 预期文件 | 测试范围 |
|---|---|---|
| Knowledge 选择、净化与信息边界 | `context_knowledge_test.go` | `TestContextToolBoundsKnowledgeForCurrentOutline`; `TestContextToolDoesNotExposeTruthUntilAfterReaderRevealChapter`; `TestContextToolExposesActiveBeliefWithoutLeakingHiddenTruth`; `TestContextToolSanitizesBeliefBoundariesByReaderCharacterAndTime`; `TestContextToolExposesReaderKnownTruthWithoutTeachingCurrentCharacter`; `TestContextToolSelectsKnowledgeForCharactersInCurrentOutline`; `TestContextToolReaderBoundary...`（若测试名后续扩展） |
| Recall/伏笔/章节与 Review 记忆 | `context_recall_test.go` | `TestContextToolSelectedMemoryRecallsStoryThreadsAndReviewLessons`; `TestContextToolSelectedMemorySurfacesAgingForeshadow`; `TestContextToolSelectedMemoryIncludesGlobalReviewLessons`; `TestContextToolKeepsFullForeshadowWhenRecallNotTriggered`; `TestContextToolFallsBackToFullForeshadowWhenSelectionIsTooSparse`; `TestContextToolLoadsArcReviewAffectingEarlierChapter`; `TestContextToolInjectsRewriteBriefForPendingRewriteChapter`; `TestContextToolOmitsRewriteBriefForNormalChapter` |
| Budget/裁剪/规范化 envelope | `context_budget_test.go` | `TestTrimByBudgetRecordsKnowledgeBoundaries`; `TestTrimByBudgetCanRemovePlatformRubricWithReferences`; `TestTrimByBudgetRemovesCanonicalMemoryKeys`; `TestTrimByBudgetKeepsStyleStats`; `TestBuildProgressStatusHidesLayeredCapacityEstimate` |
| References/平台/用户规则/文风 | `context_references_test.go` | `TestContextToolInjectsFanqieRubricOnlyWhenExplicitlySelected`; `TestContextToolUnspecifiedPlatformDoesNotLeakLoadedFanqieRubric`; `TestContextToolArchitectModeInjectsExplicitFanqieRubric`; `TestContextToolInjectsChapterTargetForArchitectAndWriter`; `TestContextToolDoesNotInjectUserDirectives`; `TestContextToolInjectsRuleViolations`; `TestContextToolInjectsStyleStats` |
| Architect/Writer envelope与SimulationProfile | `context_envelope_test.go` | `TestContextToolChapterModeIncludesWorkingAndReferenceFields`; `TestContextToolArchitectModeIncludesPlanningAndFoundation`; `TestContextToolArchitectModeIncludesFlatOutline`; `TestContextToolInjectsCompactSimulationProfile`（当前位于独立 `novel_context_simulation_test.go`，保持该文件） |
| 错误降级与核心状态 | `context_errors_test.go` | `TestContextToolWarnsWhenOptionalDataIsCorrupt`; `TestContextToolRejectsCorruptCoreState` |

说明：Context 测试的 JSON 断言不能改成字符串断言；ReaderKnown/CharacterKnown、隐藏 Truth 和 `_trimmed` 必须继续使用结构化检查。当前已有 `novel_context_simulation_test.go` 和 `novel_context_reader_boundary_test.go`，候选 4 不应为了“统一文件名”无收益地移动它们。

## 3. Import 测试

Import 当前测试按职责已经部分拆分，阶段 180 不移动文件，只补充归属映射。

| 当前文件/目标 seam | 主要测试 |
|---|---|
| `contracts_test.go` / 严格结构化契约 | `TestStructuredContractsAreStrictReady`; `TestAnalysisContractAcceptsKnowledgeActions`; `TestAnalysisContractAcceptsForeshadowLifecycleActions`; `TestCallStructuredUsesNativeSchemaWithoutPromptDuplication`; `TestCallStructuredPromptModeInjectsContract` |
| `call_test.go` / 结构化调用与错误反馈 | `TestCallStructuredNotifiesRetries`; `TestBriefErrIncludesAdapterFacts`; `TestCallStructuredCancelIsNotSemanticFailure`; `TestCallStructuredCarriesRawOnSemanticFailure`; `TestCallStructuredCarriesRawOnProtocolFailure` |
| `analyze_test.go` / 全书事实、Knowledge、批次与失效 | `TestValidateImportedFactSequence*`; `TestInvalidWorkspaceFactsReturnStateToAnalyze`; `TestValidateWorkspaceFactsInvalidatesFromFirstIllegalChapter`; `TestDiscardAnalysesAfter`; `TestBuildLedger*`; `TestValidateBatch*`; `TestAnalyzeNext*`; `TestSalvagePrefix*`; `TestAnalyzedChapters*` |
| `publish_test.go` / 映射、发布与 provenance | `TestPublishChaptersPersistsKnowledgeState`; `TestImportedFactsMappingMatchesPublishedCommitFacts`; `TestCommitArgsIncludesKnowledgeUpdates`; `TestCheckFoundationConflictsNormalizesBookMetadata`; `TestPublishChapterHandlesStalePendingCommit` |
| `runner_test.go` / Import 主流程、零污染、发布恢复 | `TestPublishRejectsInvalidFullBookFactsBeforeWritingOfficialStore`; `TestSynthesizeRejectsInvalidFullBookFactsBeforeModelCall`; `TestRunEndToEnd`; `TestRunPreservesImportedMarkdownWithoutGeneratedDraftGate`; `TestRunSetsCompletionHold`; `TestRunRejectsDifferentSource`; `TestConfirmNotesGate`; `TestStoryChoiceIgnoresStaleResolution`; `TestRunSavesFailureOnContractViolation`; `TestRunGuidanceResegments` |
| `segment_test.go` / 分章与切分 | 全部 `TestResolveSegmentation*`、`TestSegment*`、`TestPlanChunks*`、`TestChunkValidator*`、`TestPlanningBudget`、`TestBuildProjectionContextByteCap`、`TestCallStructuredTruncation` |
| `source_test.go` / 输入解码与换行 | `TestDecodeSource*`; `TestNormalizeLineEndings` |
| `state_test.go` / 工作区状态与 NextAction | `TestNextActionChain`; `TestLoadState*`; `TestIngestSnapshotConsistent`; `TestGuidanceChangeInvalidatesSegmentation`; `TestResumeSummary`; `TestResumeStatusPublishedIsTerminal`; `TestImportPreconditions` |
| `synthesize_test.go` / 综合、范围和画像 | `TestRangeDigestIdentityIgnoresFactsItDoesNotConsume`; `TestValidateStructure`; `TestAssembleFoundation*`; `TestImportedBookTitle`; `TestPlanFactRangesSplits`; `TestToCompactCarriesEvidence`; `TestSynthesizeRejectsRangeMismatch`; `TestGroupDigestsByBudget`; `TestReduceToFitMergesUntilBudget`; `TestSynthesizeDirectWithMock` |
| `workspace_test.go` / 工件与原子工作区 | 全部 `TestWorkspace*`、`TestArtifactRoundtripPreservesIdentity`、`TestDigestStableAndDistinct` |

## 4. 阶段 180 结论

- Commit、Context、Import 的 seam 可以在不改变公共接口的情况下形成清晰文件边界。
- Context 需要先保留已有的独立测试文件，不应追求机械统一。
- Import 已按职责分文件，后续主要是把 `analyze_test.go` 中的 Knowledge/全书事实测试进一步分离，拆分前必须处理共享 helper 和包级 fixture。
- 阶段 181 应先从 Commit 测试拆分开始；阶段 182 再处理 Context；阶段 183 最后处理 Import。
- 目前没有移动文件、修改测试或修改生产代码。
