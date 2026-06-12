package llm

import "context"

import "github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"

type TranslateRequest struct {
	SourceText     string
	SourceLanguage string
	TargetLanguage string
	TargetCode     string
	Model          string
	SourceType     filetype.FileType

	// SystemPrompt 由引擎统一构建并注入;为空时 provider 回退到
	// processor.BuildSystemPrompt(兼容旧调用方)。
	SystemPrompt string
}

type Provider interface {
	Translate(ctx context.Context, req TranslateRequest) (string, error)
	ListModels(ctx context.Context) ([]string, error)
	Name() string
}
