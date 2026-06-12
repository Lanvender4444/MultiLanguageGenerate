package processor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// ─────────────────────────────────────────────
// Segment 协议
//
// 结构化文档(DOCX/XLSX)不再以纯文本行对齐,而是把每个可翻译
// 单元编上 ID,以 JSON 对象 {"1":"原文",...} 发给 LLM,要求
// 返回同样 key 的 JSON。彻底消除行数错位问题。
// ─────────────────────────────────────────────

// Segment 一个可翻译单元(DOCX 的一个段落 / XLSX 的一个字符串项)
type Segment struct {
	ID   string
	Text string
}

// SegmentProcessor 支持 Segment 协议的处理器
type SegmentProcessor interface {
	Processor
	// ExtractSegments 解析源文件,返回全部可翻译单元
	ExtractSegments() ([]Segment, error)
	// RebuildSegments 用 ID→译文 映射重建文件;缺失的 ID 保留原文
	RebuildSegments(translations map[string]string, outputPath string) error
}

// MarshalSegments 把一批 segment 序列化为发给 LLM 的 JSON 对象。
// 按 ID 顺序输出,保证 LLM 看到的顺序与文档一致。
func MarshalSegments(segs []Segment) string {
	var sb strings.Builder
	sb.WriteByte('{')
	for i, s := range segs {
		if i > 0 {
			sb.WriteByte(',')
		}
		k, _ := json.Marshal(s.ID)
		v, _ := json.Marshal(s.Text)
		sb.Write(k)
		sb.WriteByte(':')
		sb.Write(v)
	}
	sb.WriteByte('}')
	return sb.String()
}

// ParseSegmentResponse 解析 LLM 返回的 JSON,校验 ID 完整性。
// 自动剥离 ```json fence 和 JSON 前后的多余文字。
func ParseSegmentResponse(raw string, want []Segment) (map[string]string, error) {
	cleaned := ExtractJSONObject(StripCodeFence(raw))
	if cleaned == "" {
		return nil, fmt.Errorf("no JSON object found in LLM response")
	}

	var m map[string]string
	if err := json.Unmarshal([]byte(cleaned), &m); err != nil {
		return nil, fmt.Errorf("parse segment JSON: %w", err)
	}

	var missing []string
	for _, s := range want {
		if v, ok := m[s.ID]; !ok || strings.TrimSpace(v) == "" {
			// 空原文对应空译文是合法的
			if strings.TrimSpace(s.Text) != "" && (!ok || strings.TrimSpace(v) == "") {
				missing = append(missing, s.ID)
			}
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		const maxShow = 10
		shown := missing
		if len(shown) > maxShow {
			shown = shown[:maxShow]
		}
		return m, fmt.Errorf("missing/empty translations for %d segment(s): %s", len(missing), strings.Join(shown, ", "))
	}
	return m, nil
}

// BatchSegments 把 segment 按字符预算切批,避免超出模型输出上限。
// 单个超长 segment 独占一批。
func BatchSegments(segs []Segment, maxChars int) [][]Segment {
	if maxChars <= 0 {
		maxChars = 4000
	}
	var batches [][]Segment
	var cur []Segment
	curLen := 0
	for _, s := range segs {
		l := len([]rune(s.Text)) + 16 // ID/引号/逗号开销
		if curLen > 0 && curLen+l > maxChars {
			batches = append(batches, cur)
			cur = nil
			curLen = 0
		}
		cur = append(cur, s)
		curLen += l
	}
	if len(cur) > 0 {
		batches = append(batches, cur)
	}
	return batches
}

// ─────────────────────────────────────────────
// LLM 输出清理
// ─────────────────────────────────────────────

var codeFenceRe = regexp.MustCompile("(?s)^\\s*```[a-zA-Z]*\\s*\n(.*?)\n?```\\s*$")

// StripCodeFence 去掉 LLM 习惯性包裹的 ```...``` 代码栅栏
func StripCodeFence(s string) string {
	if m := codeFenceRe.FindStringSubmatch(strings.TrimSpace(s)); m != nil {
		return m[1]
	}
	return s
}

// ExtractJSONObject 从文本中截取第一个完整的顶层 {...} 对象
// (LLM 偶尔会在 JSON 前后加说明文字)
func ExtractJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	inStr := false
	esc := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inStr {
			switch {
			case esc:
				esc = false
			case c == '\\':
				esc = true
			case c == '"':
				inStr = false
			}
			continue
		}
		switch c {
		case '"':
			inStr = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// ─────────────────────────────────────────────
// Run 标记:段内多格式(多 run)的处理
//
// 段落含多个 run(加粗/颜色边界)时,原文以 <r0>..</r0><r1>..</r1>
// 形式发给 LLM,由 LLM 决定译文如何分配到各 run,避免按字符
// 比例瞎切。解析失败时回退为"整段译文放第一个 run"。
// ─────────────────────────────────────────────

var runMarkRe = regexp.MustCompile(`(?s)<r(\d+)>(.*?)</r\d+>`)

// EncodeRunMarks 把多个 run 文本编码为带标记的单段文本
func EncodeRunMarks(runTexts []string) string {
	var sb strings.Builder
	for i, t := range runTexts {
		fmt.Fprintf(&sb, "<r%d>%s</r%d>", i, t, i)
	}
	return sb.String()
}

// DecodeRunMarks 从译文中解析 run 标记。
// 返回 (各 run 译文, true) — 标记完整可用;
// 返回 (nil, false) — 标记缺失/损坏,调用方应回退。
func DecodeRunMarks(translated string, runCount int) ([]string, bool) {
	matches := runMarkRe.FindAllStringSubmatch(translated, -1)
	if len(matches) == 0 {
		return nil, false
	}
	out := make([]string, runCount)
	seen := 0
	for _, m := range matches {
		idx, err := strconv.Atoi(m[1])
		if err != nil || idx < 0 || idx >= runCount {
			return nil, false
		}
		out[idx] += m[2]
		seen++
	}
	// 标记外如果还有大量裸露文字,说明 LLM 没有完整遵守协议
	stripped := runMarkRe.ReplaceAllString(translated, "")
	if len(strings.TrimSpace(stripped)) > 0 {
		return nil, false
	}
	if seen == 0 {
		return nil, false
	}
	return out, true
}

// StripRunMarks 移除所有 run 标记,只留纯文本(回退路径用)
func StripRunMarks(s string) string {
	s = runMarkRe.ReplaceAllString(s, "$2")
	// 清理残缺的孤立标记
	s = regexp.MustCompile(`</?r\d+>`).ReplaceAllString(s, "")
	return s
}

// ─────────────────────────────────────────────
// 轻量 Markdown 清理(DOCX/XLSX 译文专用,逐 segment 应用)
// ─────────────────────────────────────────────

var (
	segMdBoldRe   = regexp.MustCompile(`\*{1,3}([^*]+)\*{1,3}`)
	segMdHashRe   = regexp.MustCompile(`^#{1,6}\s+`)
	segMdCodeRe   = regexp.MustCompile("`([^`]+)`")
)

// SanitizeSegmentText 去掉 LLM 偶发输出的 Markdown 语法。
// 只做安全的最小清理,不碰 | 表格符等可能是正文的字符。
func SanitizeSegmentText(s string) string {
	s = segMdBoldRe.ReplaceAllString(s, "$1")
	s = segMdHashRe.ReplaceAllString(s, "")
	s = segMdCodeRe.ReplaceAllString(s, "$1")
	return s
}
