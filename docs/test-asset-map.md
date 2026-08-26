# 测试资产归属表

> 阶段 180—184 产物。记录当前测试函数与 seam 的归属及已完成拆分；候选 4 保持测试名称、数量、公共接口和 `-run` 筛选兼容。
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

共享构造入口仍为 `newTestContextTool`；Context 测试已按消费 seam 拆分，原文件仅保留共享 helper。

| 目标 seam | 预期文件 | 测试范围 |
|---|---|---|
| Knowledge 选择、净化与信息边界 | `context_knowledge_test.go` | `TestContextToolBoundsKnowledgeForCurrentOutline`; `TestContextToolDoesNotExposeTruthUntilAfterReaderRevealChapter`; `TestContextToolExposesActiveBeliefWithoutLeakingHiddenTruth`; `TestContextToolSanitizesBeliefBoundariesByReaderCharacterAndTime`; `TestContextToolExposesReaderKnownTruthWithoutTeachingCurrentCharacter`; `TestContextToolSelectsKnowledgeForCharactersInCurrentOutline`; `TestContextToolReaderBoundary...`（若测试名后续扩展） |
| Recall/伏笔/章节与 Review 记忆 | `context_recall_test.go` | `TestContextToolSelectedMemoryRecallsStoryThreadsAndReviewLessons`; `TestContextToolSelectedMemorySurfacesAgingForeshadow`; `TestContextToolSelectedMemoryIncludesGlobalReviewLessons`; `TestContextToolKeepsFullForeshadowWhenRecallNotTriggered`; `TestContextToolFallsBackToFullForeshadowWhenSelectionIsTooSparse`; `TestContextToolLoadsArcReviewAffectingEarlierChapter`; `TestContextToolInjectsRewriteBriefForPendingRewriteChapter`; `TestContextToolOmitsRewriteBriefForNormalChapter` |
| Budget/裁剪 | `context_budget_test.go` | `TestTrimByBudgetRecordsKnowledgeBoundaries`; `TestTrimByBudgetCanRemovePlatformRubricWithReferences`; `TestTrimByBudgetRemovesCanonicalMemoryKeys`; `TestTrimByBudgetKeepsStyleStats` |
| References/平台 | `context_references_test.go` | `TestContextToolInjectsFanqieRubricOnlyWhenExplicitlySelected`; `TestContextToolUnspecifiedPlatformDoesNotLeakLoadedFanqieRubric`; `TestContextToolArchitectModeInjectsExplicitFanqieRubric` |
| 基础模式与 UserRules/Style | `context_modes_test.go` | `TestContextToolInjectsStyleStats`; `TestContextToolChapterModeIncludesWorkingAndReferenceFields`; `TestContextToolArchitectModeIncludesPlanningAndFoundation`; `TestContextToolArchitectModeIncludesFlatOutline`; `TestContextToolInjectsChapterTargetForArchitectAndWriter` |
| Envelope 与规则事实 | `context_envelope_test.go` | `TestBuildProgressStatusHidesLayeredCapacityEstimate`; `TestContextToolDoesNotInjectUserDirectives`; `TestContextToolInjectsRuleViolations` |
| SimulationProfile | `novel_context_simulation_test.go` | `TestContextToolInjectsCompactSimulationProfile`（保留已有独立文件） |
| 错误降级与核心状态 | `context_errors_test.go` | `TestContextToolWarnsWhenOptionalDataIsCorrupt`; `TestContextToolRejectsCorruptCoreState` |

说明：Context 测试的 JSON 断言不能改成字符串断言；ReaderKnown/CharacterKnown、隐藏 Truth 和 `_trimmed` 必须继续使用结构化检查。当前已有 `novel_context_simulation_test.go` 和 `novel_context_reader_boundary_test.go`，候选 4 保留这两个独立文件，不做无收益移动。

## 3. Import 测试

Import 测试已按职责拆分；保留已有独立文件，不为拆分而拆分。

| 当前文件/目标 seam | 主要测试 |
|---|---|
| `contracts_test.go` / 严格结构化契约 | `TestStructuredContractsAreStrictReady`; `TestAnalysisContractAcceptsKnowledgeActions`; `TestAnalysisContractAcceptsForeshadowLifecycleActions`; `TestCallStructuredUsesNativeSchemaWithoutPromptDuplication`; `TestCallStructuredPromptModeInjectsContract` |
| `call_test.go` / 结构化调用与错误反馈 | `TestCallStructuredNotifiesRetries`; `TestBriefErrIncludesAdapterFacts`; `TestCallStructuredCancelIsNotSemanticFailure`; `TestCallStructuredCarriesRawOnSemanticFailure`; `TestCallStructuredCarriesRawOnProtocolFailure` |
| `analyze_test.go` / 批次分析、Salvage、预算与 Schema 失效 | `TestPlanBatchOutputBudgetCaps`; `TestPlanBatchInputBudgetCaps`; `TestValidateBatchRejections`; `TestAnalyzeNextPersistsWithRebatchOnTruncation`; `TestAnalyzeNextRejectsInvalidCumulativeSalvagePrefix`; `TestSalvagePrefixContiguous`; `TestSalvagePrefixStopsAtGap`; `TestAnalyzedChaptersInvalidatesPreviousAnalysisSchemaVersion`; `TestAnalyzedChaptersInvalidatesOnUpstreamChange` |
| `analyze_knowledge_test.go` / Knowledge 连续性与 ledger | `TestValidateImportedFactSequenceRejectsBeliefAfterLearnAcrossBatches`; `TestBuildLedger*`; `TestValidateBatchReaderReveal*`; `TestValidateBatchBeliefContinuityAndFields`; `TestValidateBatchKnowledgeContinuity` |
| `publish_provenance_test.go` / 映射、发布与 provenance | `TestPublishChaptersPersistsKnowledgeState`; `TestImportedFactsMappingMatchesPublishedCommitFacts`; `TestCommitArgsIncludesKnowledgeUpdates`; `TestCheckFoundationConflictsNormalizesBookMetadata` |
| `publish_test.go` / 发布恢复 | `TestPublishChapterHandlesStalePendingCommit` |
| `runner_recovery_test.go` / Import 主流程、零污染、发布前门禁 | `TestPublishRejectsInvalidFullBookFactsBeforeWritingOfficialStore`; `TestSynthesizeRejectsInvalidFullBookFactsBeforeModelCall`; `TestRunEndToEnd`; `TestRunPreservesImportedMarkdownWithoutGeneratedDraftGate`; `TestRunSetsCompletionHold`; `TestRunRejectsDifferentSource` |
| `runner_test.go` / Import 恢复与运行时配置 | `TestConfirmNotesGate`; `TestStoryChoiceIgnoresStaleResolution`; `TestRunSavesFailureOnContractViolation`; `TestRunGuidanceResegments`; `TestBudgetsFromDepsPerTier`; `TestBudgetsFromRuntime`; `TestProfileForKeyPolicy`; `TestCallProfileOptions` |
| `segment_test.go` / 分章与切分 | 全部 `TestResolveSegmentation*`、`TestSegment*`、`TestPlanChunks*`、`TestChunkValidator*`、`TestPlanningBudget`、`TestBuildProjectionContextByteCap`、`TestCallStructuredTruncation` |
| `source_test.go` / 输入解码与换行 | `TestDecodeSource*`; `TestNormalizeLineEndings` |
| `state_test.go` / 工作区状态与 NextAction | `TestNextActionChain`; `TestLoadState*`; `TestIngestSnapshotConsistent`; `TestGuidanceChangeInvalidatesSegmentation`; `TestResumeSummary`; `TestResumeStatusPublishedIsTerminal`; `TestImportPreconditions`（已按职责独立，无需迁移） |
| `analyze_facts_test.go` / 全书事实与工作区失效 | `TestValidateImportedFactSequenceRejectsOutOfOrderArtifactChapters`; `TestValidateImportedFactSequenceReplaysCrossBatchLifecycles`; `TestInvalidWorkspaceFactsReturnStateToAnalyze`; `TestValidateWorkspaceFactsInvalidatesFromFirstIllegalChapter`; `TestDiscardAnalysesAfter`; `TestAnalyzeNextRejectsCrossBatchFactConflictBeforeWritingArtifact` |
| `synthesize_test.go` / 综合、范围和画像 | `TestRangeDigestIdentityIgnoresFactsItDoesNotConsume`; `TestValidateStructure`; `TestAssembleFoundation*`; `TestImportedBookTitle`; `TestPlanFactRangesSplits`; `TestToCompactCarriesEvidence`; `TestSynthesizeRejectsRangeMismatch`; `TestGroupDigestsByBudget`; `TestReduceToFitMergesUntilBudget`; `TestSynthesizeDirectWithMock` |
| `workspace_test.go` / 工件与原子工作区 | 全部 `TestWorkspace*`、`TestArtifactRoundtripPreservesIdentity`、`TestDigestStableAndDistinct` |

## 4. 阶段 180 结论

- Commit、Context、Import 的 seam 可以在不改变公共接口的情况下形成清晰文件边界。
- Context 需要先保留已有的独立测试文件，不应追求机械统一。
- Import 已按职责分文件，后续主要是把 `analyze_test.go` 中的 Knowledge/全书事实测试进一步分离，拆分前必须处理共享 helper 和包级 fixture。
- 阶段 181 应先从 Commit 测试拆分开始；阶段 182 再处理 Context；阶段 183 最后处理 Import。
- 目前没有移动文件、修改测试或修改生产代码。

## 阶段 183—184 收口

- Import 主流程、事实校验、Knowledge 连续性和 Provenance 测试已迁移到职责对应文件。
- `contracts_test.go`、`call_test.go`、`segment_test.go`、`source_test.go`、`state_test.go`、`synthesize_test.go`、`workspace_test.go` 原本已按职责独立，未做无收益移动。
- 全部 Import 测试与拆分前 HEAD 保持 103 个唯一测试，未丢失、重复或新增。
- 候选 4 不修改生产代码、不新增测试框架、不改变测试语义。
