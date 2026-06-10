package llm

import "context"

type TranslateRequest struct {
	SourceText     string
	SourceLanguage string
	TargetLanguage string
	TargetCode     string
	Model          string
}

type Provider interface {
	Translate(ctx context.Context, req TranslateRequest) (string, error)
	ListModels(ctx context.Context) ([]string, error)
	Name() string
}
