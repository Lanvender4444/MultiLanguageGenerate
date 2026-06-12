package processor

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSharedStrings = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<sst xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" count="4" uniqueCount="4"><si><t>Hello World</t></si><si><r><rPr><b/></rPr><t>Bold</t></r><r><t xml:space="preserve"> normal</t></r></si><si><t>123</t></si><si><t>Product Name</t></si></sst>`

const testSheet = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1"><c r="A1" t="s"><v>0</v></c><c r="B1"><v>42.5</v></c><c r="C1" t="inlineStr"><is><t>Inline text</t></is></c></row></sheetData></worksheet>`

func writeTestXLSX(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "test.xlsx")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	entries := map[string]string{
		"[Content_Types].xml":      `<?xml version="1.0"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"/>`,
		"xl/workbook.xml":          `<?xml version="1.0"?><workbook/>`,
		"xl/sharedStrings.xml":     testSharedStrings,
		"xl/worksheets/sheet1.xml": testSheet,
	}
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	f.Close()
	return path
}

func TestXLSX_ExtractSegments(t *testing.T) {
	dir := t.TempDir()
	path := writeTestXLSX(t, dir)

	p := &XLSXProcessor{SourcePath: path}
	segs, err := p.ExtractSegments()
	if err != nil {
		t.Fatal(err)
	}

	// 期望: "Hello World", 富文本 "<r0>Bold</r0><r1> normal</r1>", "Product Name", "Inline text"
	// 纯数字 "123" 被跳过
	if len(segs) != 4 {
		t.Fatalf("expected 4 segments, got %d: %+v", len(segs), segs)
	}

	var hasRich, hasNumber bool
	for _, s := range segs {
		if strings.Contains(s.Text, "<r0>Bold</r0>") {
			hasRich = true
		}
		if s.Text == "123" {
			hasNumber = true
		}
	}
	if !hasRich {
		t.Error("rich text cell should use run marks")
	}
	if hasNumber {
		t.Error("numeric cell should be skipped")
	}
}

func TestXLSX_RebuildSegments(t *testing.T) {
	dir := t.TempDir()
	path := writeTestXLSX(t, dir)
	outPath := filepath.Join(dir, "out.xlsx")

	p := &XLSXProcessor{SourcePath: path}
	segs, err := p.ExtractSegments()
	if err != nil {
		t.Fatal(err)
	}

	translations := map[string]string{}
	for _, s := range segs {
		switch {
		case s.Text == "Hello World":
			translations[s.ID] = "你好世界"
		case strings.HasPrefix(s.Text, "<r0>"):
			translations[s.ID] = "<r0>加粗</r0><r1> 普通</r1>"
		case s.Text == "Product Name":
			translations[s.ID] = "产品名称"
		case s.Text == "Inline text":
			translations[s.ID] = "内联文字"
		}
	}

	if err := p.RebuildSegments(translations, outPath); err != nil {
		t.Fatal(err)
	}

	// 重新打开校验
	p2 := &XLSXProcessor{SourcePath: outPath}
	if err := p2.parse(); err != nil {
		t.Fatal(err)
	}

	ss := p2.cachedFiles["xl/sharedStrings.xml"]
	if !strings.Contains(ss, "<t>你好世界</t>") {
		t.Errorf("translation missing in sharedStrings: %s", ss)
	}
	if !strings.Contains(ss, "<t>加粗</t>") || !strings.Contains(ss, "<t xml:space=\"preserve\"> 普通</t>") {
		t.Errorf("rich text runs not preserved: %s", ss)
	}
	if !strings.Contains(ss, "<rPr><b/></rPr>") {
		t.Errorf("bold formatting lost: %s", ss)
	}
	if !strings.Contains(ss, "<t>123</t>") {
		t.Errorf("numeric cell should stay unchanged: %s", ss)
	}

	sheet := p2.cachedFiles["xl/worksheets/sheet1.xml"]
	if !strings.Contains(sheet, "<t>内联文字</t>") {
		t.Errorf("inline string not translated: %s", sheet)
	}
	if !strings.Contains(sheet, "<v>42.5</v>") {
		t.Errorf("numeric value damaged: %s", sheet)
	}
}

func TestXLSX_RunMarkFallback(t *testing.T) {
	dir := t.TempDir()
	path := writeTestXLSX(t, dir)
	outPath := filepath.Join(dir, "out2.xlsx")

	p := &XLSXProcessor{SourcePath: path}
	segs, err := p.ExtractSegments()
	if err != nil {
		t.Fatal(err)
	}

	// LLM 没遵守 run 标记协议 → 整段放第一个 run
	translations := map[string]string{}
	for _, s := range segs {
		if strings.HasPrefix(s.Text, "<r0>") {
			translations[s.ID] = "加粗 普通"
		}
	}
	if err := p.RebuildSegments(translations, outPath); err != nil {
		t.Fatal(err)
	}

	p2 := &XLSXProcessor{SourcePath: outPath}
	if err := p2.parse(); err != nil {
		t.Fatal(err)
	}
	ss := p2.cachedFiles["xl/sharedStrings.xml"]
	if !strings.Contains(ss, "<t>加粗 普通</t>") {
		t.Errorf("fallback failed: %s", ss)
	}
	// 第二个 run 应被清空但结构保留
	if !strings.Contains(ss, `<t xml:space="preserve"></t>`) {
		t.Errorf("second run should be emptied: %s", ss)
	}
}
