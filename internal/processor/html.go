package processor

import (
	"os"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type HTMLProcessor struct {
	SourcePath string
}

func (p *HTMLProcessor) Extract(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *HTMLProcessor) Rebuild(translatedText string, outputPath string) error {
	return os.WriteFile(outputPath, []byte(translatedText), 0644)
}

func (p *HTMLProcessor) FileType() filetype.FileType {
	return filetype.FileTypeHTML
}

func PreserveHTMLStructure(original, translated string) string {
	origLines := strings.Split(original, "\n")
	transLines := strings.Split(translated, "\n")

	if len(origLines) != len(transLines) && hasDoctype(original) {
		if len(origLines) > 0 && len(transLines) > 0 {
			if !hasDoctype(translated) && hasDoctype(original) {
				return "<!DOCTYPE html>\n" + translated
			}
		}
	}
	return translated
}

func hasDoctype(s string) bool {
	return strings.HasPrefix(strings.TrimSpace(s), "<!DOCTYPE") ||
		strings.HasPrefix(strings.TrimSpace(s), "<!doctype")
}