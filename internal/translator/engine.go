package translator

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
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

			taskCtx, cancel := context.WithTimeout(ctx, e.Timeout)
			defer cancel()

			proc := processor.NewProcessor(job.SourceFileType, job.SourceFile)
			extractedText, err := proc.Extract(job.SourceFile)
			if err != nil {
				progress <- Result{TargetCode: job.TargetCode, Error: fmt.Errorf("extract file: %w", err)}
				return
			}

			translated, err := e.Provider.Translate(taskCtx, llm.TranslateRequest{
				SourceText:     extractedText,
				SourceLanguage: job.SourceLanguage,
				TargetLanguage: job.TargetName,
				TargetCode:     job.TargetCode,
				Model:          e.Model,
				SourceType:     job.SourceFileType,
			})

			result := Result{TargetCode: job.TargetCode}
			if err != nil {
				result.Error = err
			} else {
				translated = sanitizeForDocx(translated, job.SourceFileType)
				outPath := buildOutputPath(job.SourceFile, job.TargetCode, job.OutputDir, job.SourceFileType)
				err = proc.Rebuild(translated, outPath)
				if err != nil {
					result.Error = err
				} else {
					result.OutputPath = outPath
				}
			}
			progress <- result
		}()
	}

	wg.Wait()
	close(progress)
}

func buildOutputPath(sourceFile, targetCode, outputDir string, ft filetype.FileType) string {
	ext := filepath.Ext(sourceFile)
	base := filepath.Base(sourceFile)
	name := base[:len(base)-len(ext)]

	switch ft {
	case filetype.FileTypeDOCX:
		ext = ".docx"
	case filetype.FileTypeXLSX:
		ext = ".xlsx"
	case filetype.FileTypeHTML:
	case filetype.FileTypeSRT:
	default:
	}

	outName := name + "_" + targetCode + ext

	if outputDir != "" {
		return filepath.Join(outputDir, outName)
	}
	return filepath.Join(filepath.Dir(sourceFile), outName)
}

var (
	mdBoldRe      = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`)
	mdUnderscoreRe = regexp.MustCompile(`_{1,2}([^_]+)_{1,2}`)
	mdHashRe      = regexp.MustCompile(`(?m)^#{1,6}\s+`)
	mdCodeRe      = regexp.MustCompile("`([^`]+)`")
	mdListRe      = regexp.MustCompile(`(?m)^[\s]*[-*+]\s+`)
	mdTableRe     = regexp.MustCompile(`\|[^|\n]*\|`)
	mdLinkRe      = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
)

func sanitizeForDocx(text string, ft filetype.FileType) string {
	if ft != filetype.FileTypeDOCX && ft != filetype.FileTypeXLSX {
		return text
	}
	text = mdBoldRe.ReplaceAllString(text, "$1")
	text = mdUnderscoreRe.ReplaceAllString(text, "$1")
	text = mdHashRe.ReplaceAllString(text, "")
	text = mdCodeRe.ReplaceAllString(text, "$1")
	text = mdListRe.ReplaceAllString(text, "")
	text = mdTableRe.ReplaceAllString(text, "")
	text = mdLinkRe.ReplaceAllString(text, "$1")
	return text
}