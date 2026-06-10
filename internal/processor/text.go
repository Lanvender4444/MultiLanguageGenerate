package processor

import (
	"os"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type TextProcessor struct {
	SourcePath string
	FT         filetype.FileType
}

func (p *TextProcessor) Extract(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	BOM := string([]byte{0xEF, 0xBB, 0xBF})
	content := string(data)
	if len(content) >= 3 && content[:3] == BOM {
		content = content[3:]
	}
	return content, nil
}

func (p *TextProcessor) Rebuild(translatedText string, outputPath string) error {
	return os.WriteFile(outputPath, []byte(translatedText), 0644)
}

func (p *TextProcessor) FileType() filetype.FileType {
	return p.FT
}