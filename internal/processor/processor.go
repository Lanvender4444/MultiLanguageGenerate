package processor

import (
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type Processor interface {
	Extract(filePath string) (string, error)
	Rebuild(translatedText string, outputPath string) error
	FileType() filetype.FileType
}

func NewProcessor(ft filetype.FileType, sourceFilePath string) Processor {
	switch ft {
	case filetype.FileTypeDOCX:
		return &DOCXProcessor{SourcePath: sourceFilePath}
	case filetype.FileTypeXLSX:
		return &XLSXProcessor{SourcePath: sourceFilePath}
	case filetype.FileTypeHTML:
		return &HTMLProcessor{SourcePath: sourceFilePath}
	default:
		return &TextProcessor{SourcePath: sourceFilePath, FT: ft}
	}
}
