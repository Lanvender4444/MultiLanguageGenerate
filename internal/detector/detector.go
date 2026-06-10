package detector

import (
	"context"
	"unicode"
	"unicode/utf8"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
)

func DetectLocal(content string) (code, name string) {
	if len(content) > 512 {
		content = content[:512]
	}

	var cjkCount, latinCount, cyrillicCount, arabicCount, devanagariCount int
	for _, r := range content {
		if unicode.Is(unicode.Han, r) {
			cjkCount++
		} else if unicode.Is(unicode.Latin, r) {
			latinCount++
		} else if unicode.Is(unicode.Cyrillic, r) {
			cyrillicCount++
		} else if unicode.Is(unicode.Arabic, r) {
			arabicCount++
		} else if unicode.Is(unicode.Devanagari, r) {
			devanagariCount++
		}
	}

	total := utf8.RuneCountInString(content)
	if total == 0 {
		return "en", "English"
	}

	switch {
	case cjkCount*100/total > 30:
		return "zh-CN", "中文（简体）"
	case cyrillicCount*100/total > 30:
		return "ru", "русский"
	case arabicCount*100/total > 30:
		return "ar", "العربية"
	case devanagariCount*100/total > 30:
		return "hi", "हिन्दी"
	default:
		return "en", "English"
	}
}

func DetectAI(ctx context.Context, content string, provider llm.Provider, model string) (code, name string, err error) {
	if len(content) > 200 {
		content = content[:200]
	}

	result, err := provider.Translate(ctx, llm.TranslateRequest{
		SourceText:     content,
		SourceLanguage: "auto-detect",
		TargetLanguage: "return only the language code and name",
		TargetCode:     "detect",
		Model:          model,
		SourceType:     filetype.FileTypePlainText,
	})
	if err != nil {
		return "", "", err
	}

	return "auto", result, nil
}
