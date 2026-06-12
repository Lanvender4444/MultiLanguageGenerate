package processor

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

// ─────────────────────────────────────────────
// 翻译输出结构校验
//
// HTML/XML/JSON 等结构化文本走整文件翻译,LLM 可能破坏结构。
// 翻译后调用 ValidateTranslation,失败则引擎自动重试一次。
// ─────────────────────────────────────────────

// ValidateTranslation 校验译文结构是否完好。返回 nil 表示通过。
func ValidateTranslation(ft filetype.FileType, original, translated string) error {
	if strings.TrimSpace(translated) == "" {
		return fmt.Errorf("translation is empty")
	}

	switch ft {
	case filetype.FileTypeJSON:
		return validateJSON(original, translated)
	case filetype.FileTypeXML:
		return validateXML(translated)
	case filetype.FileTypeHTML:
		return validateHTML(original, translated)
	case filetype.FileTypeSRT:
		return validateSRT(original, translated)
	case filetype.FileTypeCSV:
		return validateCSV(original, translated)
	default:
		return nil
	}
}

// ── JSON:必须可解析,且 key 集合不变 ──

func validateJSON(original, translated string) error {
	if !json.Valid([]byte(translated)) {
		return fmt.Errorf("translated output is not valid JSON")
	}
	origKeys := collectJSONKeys(original)
	transKeys := collectJSONKeys(translated)
	if len(origKeys) != len(transKeys) {
		return fmt.Errorf("JSON key count changed: %d -> %d", len(origKeys), len(transKeys))
	}
	for k := range origKeys {
		if !transKeys[k] {
			return fmt.Errorf("JSON key %q missing in translation", k)
		}
	}
	return nil
}

func collectJSONKeys(s string) map[string]bool {
	keys := map[string]bool{}
	var walk func(v interface{}, path string)
	walk = func(v interface{}, path string) {
		switch t := v.(type) {
		case map[string]interface{}:
			for k, child := range t {
				p := path + "." + k
				keys[p] = true
				walk(child, p)
			}
		case []interface{}:
			for i, child := range t {
				walk(child, fmt.Sprintf("%s[%d]", path, i))
			}
		}
	}
	var v interface{}
	if err := json.Unmarshal([]byte(s), &v); err == nil {
		walk(v, "")
	}
	return keys
}

// ── XML:必须 well-formed ──

func validateXML(translated string) error {
	dec := xml.NewDecoder(strings.NewReader(translated))
	dec.Strict = false
	for {
		_, err := dec.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("translated XML is not well-formed: %w", err)
		}
	}
}

// ── HTML:标签序列必须一致 ──

var htmlTagRe = regexp.MustCompile(`(?s)<(/?)([a-zA-Z][a-zA-Z0-9-]*)[^>]*>`)

func validateHTML(original, translated string) error {
	origTags := extractTagSequence(original)
	transTags := extractTagSequence(translated)
	if len(origTags) != len(transTags) {
		return fmt.Errorf("HTML tag count changed: %d -> %d", len(origTags), len(transTags))
	}
	for i := range origTags {
		if origTags[i] != transTags[i] {
			return fmt.Errorf("HTML tag sequence diverges at #%d: %q -> %q", i, origTags[i], transTags[i])
		}
	}
	return nil
}

func extractTagSequence(s string) []string {
	var tags []string
	for _, m := range htmlTagRe.FindAllStringSubmatch(s, -1) {
		tags = append(tags, m[1]+strings.ToLower(m[2]))
	}
	return tags
}

// ── SRT:块数与时间轴必须一致 ──

var srtTimeRe = regexp.MustCompile(`\d{2}:\d{2}:\d{2}[,.]\d{3}\s*-->\s*\d{2}:\d{2}:\d{2}[,.]\d{3}`)

func validateSRT(original, translated string) error {
	origTimes := srtTimeRe.FindAllString(original, -1)
	transTimes := srtTimeRe.FindAllString(translated, -1)
	if len(origTimes) != len(transTimes) {
		return fmt.Errorf("subtitle block count changed: %d -> %d", len(origTimes), len(transTimes))
	}
	for i := range origTimes {
		if normalizeSpace(origTimes[i]) != normalizeSpace(transTimes[i]) {
			return fmt.Errorf("timestamp #%d modified: %q -> %q", i, origTimes[i], transTimes[i])
		}
	}
	return nil
}

func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// ── CSV:行数必须一致 ──

func validateCSV(original, translated string) error {
	o := strings.Count(strings.TrimRight(original, "\r\n"), "\n")
	t := strings.Count(strings.TrimRight(translated, "\r\n"), "\n")
	if o != t {
		return fmt.Errorf("CSV line count changed: %d -> %d", o+1, t+1)
	}
	return nil
}
