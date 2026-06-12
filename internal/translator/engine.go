package translator

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/processor"
)

type Engine struct {
	Provider   llm.Provider
	MaxWorkers int
	Timeout    time.Duration
	Model      string

	// BatchChars Segment 协议单批最大字符数(防止超出模型输出上限)
	BatchChars int
	// MaxRetries 校验失败后的重试次数
	MaxRetries int
}

func NewEngine(provider llm.Provider, model string, maxWorkers int, timeout time.Duration) *Engine {
	if maxWorkers <= 0 {
		maxWorkers = 5
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Engine{
		Provider:   provider,
		MaxWorkers: maxWorkers,
		Timeout:    timeout,
		Model:      model,
		BatchChars: 4000,
		MaxRetries: 1,
	}
}

func (e *Engine) Run(ctx context.Context, jobs []Job, progress chan<- Result) {
	sem := make(chan struct{}, e.MaxWorkers)
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		job := job
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := Result{TargetCode: job.TargetCode}
			outPath := buildOutputPath(job.SourceFile, job.TargetCode, job.OutputDir, job.SourceFileType)

			proc := processor.NewProcessor(job.SourceFileType, job.SourceFile)

			var err error
			if sp, ok := proc.(processor.SegmentProcessor); ok {
				// DOCX/XLSX:Segment JSON 协议,分批翻译,ID 对位回填
				err = e.translateSegments(ctx, sp, job, outPath)
			} else {
				// 其余类型:整文件翻译 + 结构校验 + 自动重试
				err = e.translateWholeFile(ctx, proc, job, outPath)
			}

			if err != nil {
				result.Error = err
			} else {
				result.OutputPath = outPath
			}
			progress <- result
		}()
	}

	wg.Wait()
	close(progress)
}

// ─────────────────────────────────────────────
// Segment 协议流程(DOCX/XLSX)
// ─────────────────────────────────────────────

func (e *Engine) translateSegments(ctx context.Context, sp processor.SegmentProcessor, job Job, outPath string) error {
	segs, err := sp.ExtractSegments()
	if err != nil {
		return fmt.Errorf("extract segments: %w", err)
	}
	if len(segs) == 0 {
		return fmt.Errorf("no translatable text found in %s", filepath.Base(job.SourceFile))
	}

	sysPrompt := processor.BuildSegmentSystemPrompt(job.SourceLanguage, job.TargetName, job.TargetCode, job.SourceFileType)
	batches := processor.BatchSegments(segs, e.BatchChars)
	translations := make(map[string]string, len(segs))

	for bi, batch := range batches {
		m, err := e.translateBatch(ctx, batch, sysPrompt, job)
		if err != nil {
			return fmt.Errorf("batch %d/%d: %w", bi+1, len(batches), err)
		}
		for k, v := range m {
			translations[k] = v
		}
	}

	if err := sp.RebuildSegments(translations, outPath); err != nil {
		return fmt.Errorf("rebuild file: %w", err)
	}
	return nil
}

// translateBatch 翻译一批 segment,解析失败自动重试
func (e *Engine) translateBatch(ctx context.Context, batch []processor.Segment, sysPrompt string, job Job) (map[string]string, error) {
	payload := processor.MarshalSegments(batch)

	var lastErr error
	for attempt := 0; attempt <= e.MaxRetries; attempt++ {
		prompt := sysPrompt
		if attempt > 0 {
			prompt += fmt.Sprintf("\n\nIMPORTANT: your previous reply was rejected (%v). Reply with ONLY a valid JSON object containing exactly the same keys as the input, no code fences, no extra text.", lastErr)
		}

		callCtx, cancel := context.WithTimeout(ctx, e.Timeout)
		raw, err := e.Provider.Translate(callCtx, llm.TranslateRequest{
			SourceText:     payload,
			SourceLanguage: job.SourceLanguage,
			TargetLanguage: job.TargetName,
			TargetCode:     job.TargetCode,
			Model:          e.Model,
			SourceType:     job.SourceFileType,
			SystemPrompt:   prompt,
		})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}

		m, perr := processor.ParseSegmentResponse(raw, batch)
		if perr == nil {
			return m, nil
		}
		// 部分缺失时保留已有结果继续重试;最后一次重试后若仍缺失,
		// 用已解析的部分(缺失段落保留原文,不至于整个文件失败)
		lastErr = perr
		if attempt == e.MaxRetries && m != nil {
			return m, nil
		}
	}
	return nil, lastErr
}

// ─────────────────────────────────────────────
// 整文件流程(Markdown/HTML/XML/JSON/CSV/SRT/PO/纯文本)
// ─────────────────────────────────────────────

func (e *Engine) translateWholeFile(ctx context.Context, proc processor.Processor, job Job, outPath string) error {
	extracted, err := proc.Extract(job.SourceFile)
	if err != nil {
		return fmt.Errorf("extract file: %w", err)
	}

	sysPrompt := processor.BuildSystemPrompt(job.SourceLanguage, job.TargetName, job.TargetCode, job.SourceFileType)

	var lastErr error
	for attempt := 0; attempt <= e.MaxRetries; attempt++ {
		prompt := sysPrompt
		if attempt > 0 {
			prompt += fmt.Sprintf("\n\nIMPORTANT: your previous translation was rejected because it broke the file structure (%v). Re-translate and keep the structure EXACTLY identical to the input.", lastErr)
		}

		callCtx, cancel := context.WithTimeout(ctx, e.Timeout)
		translated, err := e.Provider.Translate(callCtx, llm.TranslateRequest{
			SourceText:     extracted,
			SourceLanguage: job.SourceLanguage,
			TargetLanguage: job.TargetName,
			TargetCode:     job.TargetCode,
			Model:          e.Model,
			SourceType:     job.SourceFileType,
			SystemPrompt:   prompt,
		})
		cancel()
		if err != nil {
			lastErr = err
			continue
		}

		translated = processor.StripCodeFence(translated)

		if verr := processor.ValidateTranslation(job.SourceFileType, extracted, translated); verr != nil {
			lastErr = verr
			continue
		}

		if err := proc.Rebuild(translated, outPath); err != nil {
			return fmt.Errorf("rebuild file: %w", err)
		}
		return nil
	}
	return fmt.Errorf("translation failed after %d attempt(s): %w", e.MaxRetries+1, lastErr)
}

// ─────────────────────────────────────────────

func buildOutputPath(sourceFile, targetCode, outputDir string, ft filetype.FileType) string {
	ext := filepath.Ext(sourceFile)
	base := filepath.Base(sourceFile)
	name := base[:len(base)-len(ext)]

	switch ft {
	case filetype.FileTypeDOCX:
		ext = ".docx"
	case filetype.FileTypeXLSX:
		ext = ".xlsx"
	}

	outName := name + "_" + targetCode + ext

	if outputDir != "" {
		return filepath.Join(outputDir, outName)
	}
	return filepath.Join(filepath.Dir(sourceFile), outName)
}
