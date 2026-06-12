package processor

import (
	"strings"
	"testing"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

func TestMarshalAndParseSegments(t *testing.T) {
	segs := []Segment{
		{ID: "0", Text: "Hello \"world\""},
		{ID: "3", Text: "Line1\nLine2"},
	}
	payload := MarshalSegments(segs)

	// 模拟 LLM 返回(带 code fence 和前置说明)
	raw := "Here is the translation:\n```json\n{\"0\":\"你好\\\"世界\\\"\",\"3\":\"第一行\\n第二行\"}\n```"
	m, err := ParseSegmentResponse(raw, segs)
	if err != nil {
		t.Fatalf("parse: %v (payload=%s)", err, payload)
	}
	if m["0"] != `你好"世界"` {
		t.Errorf("seg0=%q", m["0"])
	}
	if m["3"] != "第一行\n第二行" {
		t.Errorf("seg3=%q", m["3"])
	}
}

func TestParseSegmentResponse_MissingID(t *testing.T) {
	segs := []Segment{{ID: "0", Text: "a"}, {ID: "1", Text: "b"}}
	_, err := ParseSegmentResponse(`{"0":"x"}`, segs)
	if err == nil {
		t.Fatal("expected error for missing ID")
	}
	if !strings.Contains(err.Error(), "1") {
		t.Errorf("error should mention missing ID: %v", err)
	}
}

func TestBatchSegments(t *testing.T) {
	segs := []Segment{
		{ID: "0", Text: strings.Repeat("a", 100)},
		{ID: "1", Text: strings.Repeat("b", 100)},
		{ID: "2", Text: strings.Repeat("c", 100)},
	}
	batches := BatchSegments(segs, 150)
	if len(batches) != 3 {
		t.Fatalf("expected 3 batches, got %d", len(batches))
	}
	// 超长 segment 独占一批且不丢失
	total := 0
	for _, b := range batches {
		total += len(b)
	}
	if total != 3 {
		t.Errorf("segments lost in batching: %d", total)
	}
}

func TestRunMarks_RoundTrip(t *testing.T) {
	encoded := EncodeRunMarks([]string{"Hello ", "bold", " world"})
	if encoded != "<r0>Hello </r0><r1>bold</r1><r2> world</r2>" {
		t.Fatalf("encoded=%q", encoded)
	}

	texts, ok := DecodeRunMarks("<r0>你好 </r0><r1>加粗</r1><r2> 世界</r2>", 3)
	if !ok {
		t.Fatal("decode should succeed")
	}
	if texts[1] != "加粗" {
		t.Errorf("run1=%q", texts[1])
	}
}

func TestDecodeRunMarks_Fallback(t *testing.T) {
	// LLM 没输出标记 → 应触发回退
	if _, ok := DecodeRunMarks("你好加粗世界", 3); ok {
		t.Error("plain text should not decode as run marks")
	}
	// 标记外有裸文字 → 协议被破坏,回退
	if _, ok := DecodeRunMarks("<r0>你好</r0>多余文字", 1); ok {
		t.Error("stray text outside marks should fail")
	}
	// 越界标记 → 回退
	if _, ok := DecodeRunMarks("<r9>你好</r9>", 2); ok {
		t.Error("out-of-range mark should fail")
	}
}

func TestStripCodeFence(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	if got := StripCodeFence(in); got != `{"a":1}` {
		t.Errorf("got %q", got)
	}
	// 无 fence 原样返回
	if got := StripCodeFence(`{"a":1}`); got != `{"a":1}` {
		t.Errorf("got %q", got)
	}
}

func TestSanitizeSegmentText(t *testing.T) {
	if got := SanitizeSegmentText("**加粗** 和 `代码`"); got != "加粗 和 代码" {
		t.Errorf("got %q", got)
	}
	// 正文里的 | 和 - 不应被破坏
	if got := SanitizeSegmentText("A | B - C"); got != "A | B - C" {
		t.Errorf("got %q", got)
	}
}

func TestValidateTranslation(t *testing.T) {
	// JSON
	orig := `{"title":"Hello","n":1}`
	if err := ValidateTranslation(filetype.FileTypeJSON, orig, `{"title":"你好","n":1}`); err != nil {
		t.Errorf("valid JSON rejected: %v", err)
	}
	if err := ValidateTranslation(filetype.FileTypeJSON, orig, `{"title":"你好",}`); err == nil {
		t.Error("broken JSON accepted")
	}
	if err := ValidateTranslation(filetype.FileTypeJSON, orig, `{"标题":"你好","n":1}`); err == nil {
		t.Error("translated key accepted")
	}

	// XML
	if err := ValidateTranslation(filetype.FileTypeXML, "<a><b>x</b></a>", "<a><b>是</b></a>"); err != nil {
		t.Errorf("valid XML rejected: %v", err)
	}
	if err := ValidateTranslation(filetype.FileTypeXML, "<a><b>x</b></a>", "<a><b>是</a>"); err == nil {
		t.Error("malformed XML accepted")
	}

	// HTML 标签序列
	if err := ValidateTranslation(filetype.FileTypeHTML, `<p class="x">Hi</p>`, `<p class="x">你好</p>`); err != nil {
		t.Errorf("valid HTML rejected: %v", err)
	}
	if err := ValidateTranslation(filetype.FileTypeHTML, `<p>Hi</p>`, `<div>你好</div>`); err == nil {
		t.Error("changed tag accepted")
	}

	// SRT 时间轴
	srt := "1\n00:00:01,000 --> 00:00:02,000\nHello\n"
	if err := ValidateTranslation(filetype.FileTypeSRT, srt, "1\n00:00:01,000 --> 00:00:02,000\n你好\n"); err != nil {
		t.Errorf("valid SRT rejected: %v", err)
	}
	if err := ValidateTranslation(filetype.FileTypeSRT, srt, "1\n00:00:01,000 --> 00:00:05,000\n你好\n"); err == nil {
		t.Error("modified timestamp accepted")
	}
}
