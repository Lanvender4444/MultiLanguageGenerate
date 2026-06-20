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
	// Glossary 为该目标语言渲染好的"专业名词"约束块；非空时追加到系统提示。
	Glossary string
}

type Result struct {
	TargetCode     string
	OutputPath     string
	TranslatedText string
	Error          error
}