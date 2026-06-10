package translator

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/yourname/MultiLanguageGenerate/internal/llm"
	"github.com/yourname/MultiLanguageGenerate/internal/output"
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

			translated, err := e.Provider.Translate(taskCtx, llm.TranslateRequest{
				SourceText:     job.SourceText,
				SourceLanguage: job.SourceLanguage,
				TargetLanguage: job.TargetName,
				TargetCode:     job.TargetCode,
				Model:          e.Model,
			})

			result := Result{TargetCode: job.TargetCode}
			if err != nil {
				result.Error = err
			} else {
				outPath := output.BuildOutputPath(job.SourceFile, job.TargetCode, job.OutputDir)
				err = os.WriteFile(outPath, []byte(translated), 0644)
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
