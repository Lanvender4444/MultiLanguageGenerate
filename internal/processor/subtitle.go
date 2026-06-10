package processor

import (
	"os"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type SRTProcessor struct {
	SourcePath string
}

func (p *SRTProcessor) Extract(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *SRTProcessor) Rebuild(translatedText string, outputPath string) error {
	return os.WriteFile(outputPath, []byte(translatedText), 0644)
}

func (p *SRTProcessor) GetFileType() filetype.FileType {
	return filetype.FileTypeSRT
}

func IsTimeLine(line string) bool {
	return strings.Contains(line, "-->") && strings.Count(line, ":") >= 2
}

func IsSequenceNumber(line string) bool {
	line = strings.TrimSpace(line)
	if len(line) == 0 {
		return false
	}
	for _, c := range line {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}