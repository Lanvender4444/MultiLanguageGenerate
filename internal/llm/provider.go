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
}

type Provider interface {
	Translate(ctx context.Context, req TranslateRequest) (string, error)
	ListModels(ctx context.Context) ([]string, error)
	Name() string
}