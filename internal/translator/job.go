package translator

import (
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type Job struct {
	SourceText     string
	SourceFile     string
	SourceLanguage string
	TargetCode     string
	TargetName     string
	OutputDir      string
	SourceFileType filetype.FileType
	SkipOutput     bool
}

type Result struct {
	TargetCode     string
	OutputPath     string
	TranslatedText string
	Error          error
}