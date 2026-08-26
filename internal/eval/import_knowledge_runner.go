package eval

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CalibrationSample 是一条独立的 Import 认知动作校准样本。
// 文本本身由调用方管理；Runner 只负责调度和保存脱敏结果。
type CalibrationSample struct {
	ID       string
	Category string
}

// KnowledgeActionResult 是模型对单条样本输出的动作级脱敏结果。
type KnowledgeActionResult struct {
	ID        string `json:"id"`
	Action    string `json:"action"`
	Character string `json:"character,omitempty"`
	Belief    bool   `json:"belief,omitempty"`
}

// ImportKnowledgeSampleResult 是单条校准样本的脱敏结果。
type ImportKnowledgeSampleResult struct {
	SampleID     string                  `json:"sample_id"`
	Category     string                  `json:"category"`
	PromptName   string                  `json:"prompt_name"`
	PromptDigest string                  `json:"prompt_digest"`
	Updates      []KnowledgeActionResult `json:"updates"`
	Usage        UsageMetrics            `json:"usage"`
}

// ImportKnowledgeErrorRecord 只保存可诊断的错误摘要，不保存原始响应。
type ImportKnowledgeErrorRecord struct {
	SampleID     string `json:"sample_id"`
	PromptName   string `json:"prompt_name"`
	PromptDigest string `json:"prompt_digest"`
	Error        string `json:"error"`
}

// ImportKnowledgeRunnerOptions 标识一次 baseline 或 calibrated 运行。
type ImportKnowledgeRunnerOptions struct {
	PromptName   string
	PromptDigest string
}

// ImportKnowledgeRunnerSummary 是一次可断点评测的聚合状态。
type ImportKnowledgeRunnerSummary struct {
	Completed int
	Failed    int
}

var ErrImportKnowledgeResultInvalid = errors.New("import knowledge calibration result invalid")

// ImportKnowledgeRunner 按样本保存校准结果。它不是运行时评测框架，只服务于离线校准。
type ImportKnowledgeRunner struct {
	dir     string
	samples []CalibrationSample
	opts    ImportKnowledgeRunnerOptions
}

func NewImportKnowledgeRunner(dir string, samples []CalibrationSample, opts ImportKnowledgeRunnerOptions) ImportKnowledgeRunner {
	return ImportKnowledgeRunner{dir: dir, samples: samples, opts: opts}
}

// Run 执行尚未成功完成的样本。已存在且指纹匹配的成功结果会跳过；失败记录不会阻塞后续样本。
// 任一失败都会返回聚合 error，但已成功的样本仍然落盘，便于下一次续跑。
func (r ImportKnowledgeRunner) Run(execute func(CalibrationSample) (ImportKnowledgeSampleResult, error)) (ImportKnowledgeRunnerSummary, error) {
	if strings.TrimSpace(r.dir) == "" {
		return ImportKnowledgeRunnerSummary{}, fmt.Errorf("import knowledge runner: missing output directory")
	}
	if strings.TrimSpace(r.opts.PromptName) == "" || strings.TrimSpace(r.opts.PromptDigest) == "" {
		return ImportKnowledgeRunnerSummary{}, fmt.Errorf("import knowledge runner: missing prompt identity")
	}
	if execute == nil {
		return ImportKnowledgeRunnerSummary{}, fmt.Errorf("import knowledge runner: missing executor")
	}
	if err := os.MkdirAll(filepath.Join(r.dir, "results"), 0o755); err != nil {
		return ImportKnowledgeRunnerSummary{}, fmt.Errorf("create result directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Join(r.dir, "errors"), 0o755); err != nil {
		return ImportKnowledgeRunnerSummary{}, fmt.Errorf("create error directory: %w", err)
	}

	var summary ImportKnowledgeRunnerSummary
	var failures []error
	for _, sample := range r.samples {
		if err := validateCalibrationSample(sample); err != nil {
			failures = append(failures, err)
			continue
		}
		resultPath := filepath.Join(r.dir, "results", sample.ID+".json")
		if ok, err := r.hasMatchingResult(resultPath, sample); err != nil {
			failures = append(failures, fmt.Errorf("%s: %w", sample.ID, err))
			continue
		} else if ok {
			summary.Completed++
			continue
		}

		result, err := execute(sample)
		if err != nil {
			summary.Failed++
			failures = append(failures, fmt.Errorf("%s: %w", sample.ID, err))
			record := ImportKnowledgeErrorRecord{
				SampleID: sample.ID, PromptName: r.opts.PromptName,
				PromptDigest: r.opts.PromptDigest, Error: err.Error(),
			}
			if writeErr := writeJSONAtomic(filepath.Join(r.dir, "errors", sample.ID+".json"), record); writeErr != nil {
				failures = append(failures, fmt.Errorf("%s error record: %w", sample.ID, writeErr))
			}
			continue
		}
		if result.SampleID != sample.ID || result.Category != sample.Category {
			summary.Failed++
			err := fmt.Errorf("%w: sample=%q result=%q/%q", ErrImportKnowledgeResultInvalid, sample.ID, result.SampleID, result.Category)
			failures = append(failures, err)
			continue
		}
		result.PromptName = r.opts.PromptName
		result.PromptDigest = r.opts.PromptDigest
		if err := writeJSONAtomic(resultPath, result); err != nil {
			summary.Failed++
			failures = append(failures, fmt.Errorf("%s result: %w", sample.ID, err))
			continue
		}
		_ = os.Remove(filepath.Join(r.dir, "errors", sample.ID+".json"))
		summary.Completed++
	}
	if len(failures) > 0 {
		return summary, fmt.Errorf("import knowledge calibration run failed: %w", errors.Join(failures...))
	}
	return summary, nil
}

func validateCalibrationSample(sample CalibrationSample) error {
	if strings.TrimSpace(sample.ID) == "" || strings.ContainsAny(sample.ID, `/\\`) || sample.ID == "." || sample.ID == ".." {
		return fmt.Errorf("%w: invalid sample id %q", ErrImportKnowledgeResultInvalid, sample.ID)
	}
	if strings.TrimSpace(sample.Category) == "" {
		return fmt.Errorf("%w: sample %q has empty category", ErrImportKnowledgeResultInvalid, sample.ID)
	}
	return nil
}

func (r ImportKnowledgeRunner) hasMatchingResult(path string, sample CalibrationSample) (bool, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var result ImportKnowledgeSampleResult
	if err := json.Unmarshal(data, &result); err != nil {
		return false, fmt.Errorf("decode result: %w", err)
	}
	return result.SampleID == sample.ID && result.Category == sample.Category &&
		result.PromptName == r.opts.PromptName && result.PromptDigest == r.opts.PromptDigest, nil
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	name := filepath.Base(path)
	hash := sha256.Sum256(data)
	tmp := filepath.Join(filepath.Dir(path), "."+name+"."+hex.EncodeToString(hash[:])[:12]+".tmp")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}
