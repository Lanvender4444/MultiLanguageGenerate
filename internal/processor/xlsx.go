package processor

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

// ─────────────────────────────────────────────
// XLSXProcessor
//
// XLSX 的文字存在两处:
//   1. xl/sharedStrings.xml 的 <si>...</si>(绝大多数单元格)
//   2. xl/worksheets/*.xml 的 <is>...</is>(内联字符串单元格)
// 只改这些 XML 里 <t> 节点的文字,其余字节原样保留,
// 数字、公式、样式、图表全部不受影响。
// ─────────────────────────────────────────────

type XLSXProcessor struct {
	SourcePath string

	cachedFiles map[string]string      // 需要改写的 XML 文件内容
	cachedBlocks []xlsxBlock           // 全部字符串块,下标即 Segment ID
}

// xlsxBlock 一个 <si> 或 <is> 块
type xlsxBlock struct {
	File       string    // 所属 zip 内文件名
	StartInXML int       // 块内容(不含外层标签)在文件中的起始偏移
	EndInXML   int       // 块内容结束偏移(不含)
	Content    string    // 块内容原始 XML
	Runs       []RunInfo // 块内所有 <t> 节点(偏移相对 Content)
	FullText   string
	Translatable bool
}

// 标签允许命名空间前缀(<si> 或 <x:si>),兼容非 Excel 写入器
var (
	siBlockRe = regexp.MustCompile(`(?s)<(?:\w+:)?si(?:\s[^>]*)?>(.*?)</(?:\w+:)?si>`)
	isBlockRe = regexp.MustCompile(`(?s)<(?:\w+:)?is(?:\s[^>]*)?>(.*?)</(?:\w+:)?is>`)
	// 注意:必须排除自闭合 <t/>、<t a="1"/>,否则会把后续 XML 吞进文本组
	xlsxTRe   = regexp.MustCompile(`(?s)(<(?:\w+:)?t(?:\s+[^>]*[^>/\s])?\s*>)(.*?)(</(?:\w+:)?t>)`)
	rPhRe     = regexp.MustCompile(`(?s)<(?:\w+:)?rPh(?:\s[^>]*)?>.*?</(?:\w+:)?rPh>`)
)

func (p *XLSXProcessor) FileType() filetype.FileType {
	return filetype.FileTypeXLSX
}

// ─────────────────────────────────────────────
// 解析
// ─────────────────────────────────────────────

func (p *XLSXProcessor) parse() error {
	if p.cachedFiles != nil {
		return nil
	}

	r, err := zip.OpenReader(p.SourcePath)
	if err != nil {
		return fmt.Errorf("open xlsx zip %q: %w", p.SourcePath, err)
	}
	defer r.Close()

	files := map[string]string{}
	var names []string
	for _, f := range r.File {
		isShared := f.Name == "xl/sharedStrings.xml"
		isSheet := strings.HasPrefix(f.Name, "xl/worksheets/") && strings.HasSuffix(f.Name, ".xml")
		if !isShared && !isSheet {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %q: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read %q: %w", f.Name, err)
		}
		files[f.Name] = string(data)
		names = append(names, f.Name)
	}
	sort.Strings(names) // 解析顺序稳定,保证 Segment ID 可复现

	var blocks []xlsxBlock
	for _, name := range names {
		content := files[name]
		blockRe := isBlockRe
		if name == "xl/sharedStrings.xml" {
			blockRe = siBlockRe
		}
		for _, loc := range blockRe.FindAllStringSubmatchIndex(content, -1) {
			inner := content[loc[2]:loc[3]]
			blocks = append(blocks, parseXLSXBlock(name, loc[2], loc[3], inner))
		}
	}

	p.cachedFiles = files
	p.cachedBlocks = blocks
	return nil
}

func parseXLSXBlock(file string, start, end int, inner string) xlsxBlock {
	// 标出 <rPh>(拼音注音)范围,其中的 <t> 不参与翻译
	phonetic := rPhRe.FindAllStringIndex(inner, -1)
	inPhonetic := func(s, e int) bool {
		for _, ph := range phonetic {
			if s >= ph[0] && e <= ph[1] {
				return true
			}
		}
		return false
	}

	var runs []RunInfo
	var sb strings.Builder
	for _, m := range xlsxTRe.FindAllStringSubmatchIndex(inner, -1) {
		if inPhonetic(m[0], m[1]) {
			continue
		}
		text := decodeXMLEntities(inner[m[4]:m[5]])
		runs = append(runs, RunInfo{
			OpenTag:     inner[m[2]:m[3]],
			CloseTag:    inner[m[6]:m[7]],
			Text:        text,
			StartInPara: m[0],
			EndInPara:   m[1],
		})
		sb.WriteString(text)
	}

	full := sb.String()
	return xlsxBlock{
		File:         file,
		StartInXML:   start,
		EndInXML:     end,
		Content:      inner,
		Runs:         runs,
		FullText:     full,
		Translatable: isTranslatableCell(full),
	}
}

// isTranslatableCell 纯数字/符号/空白的单元格不发给 LLM
func isTranslatableCell(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	for _, r := range s {
		if unicode.IsLetter(r) {
			return true
		}
	}
	return false
}

// ─────────────────────────────────────────────
// Segment 协议
// ─────────────────────────────────────────────

func (p *XLSXProcessor) ExtractSegments() ([]Segment, error) {
	if err := p.parse(); err != nil {
		return nil, err
	}
	if len(p.cachedBlocks) == 0 {
		names := make([]string, 0, len(p.cachedFiles))
		for n := range p.cachedFiles {
			names = append(names, n)
		}
		sort.Strings(names)
		return nil, fmt.Errorf("xlsx: no string blocks (<si>/<is>) found; scanned %d file(s): %s", len(names), strings.Join(names, ", "))
	}
	var segs []Segment
	for i, b := range p.cachedBlocks {
		if !b.Translatable {
			continue
		}
		text := b.FullText
		if countTextRuns(b.Runs) > 1 {
			runTexts := make([]string, len(b.Runs))
			for j, r := range b.Runs {
				runTexts[j] = r.Text
			}
			text = EncodeRunMarks(runTexts)
		}
		segs = append(segs, Segment{ID: strconv.Itoa(i), Text: text})
	}
	return segs, nil
}

func (p *XLSXProcessor) RebuildSegments(translations map[string]string, outputPath string) error {
	if err := p.parse(); err != nil {
		return err
	}

	// 每个块算出新内容
	newContents := make([]string, len(p.cachedBlocks))
	for i, b := range p.cachedBlocks {
		newContents[i] = rebuildXLSXBlock(b, translations[strconv.Itoa(i)])
	}

	// 按文件分组,偏移拼接重建
	newFiles := map[string]string{}
	for name, content := range p.cachedFiles {
		var sb strings.Builder
		sb.Grow(len(content))
		pos := 0
		for i, b := range p.cachedBlocks {
			if b.File != name {
				continue
			}
			sb.WriteString(content[pos:b.StartInXML])
			sb.WriteString(newContents[i])
			pos = b.EndInXML
		}
		sb.WriteString(content[pos:])
		newFiles[name] = sb.String()
	}

	return rebuildZipWithFiles(p.SourcePath, newFiles, outputPath)
}

// rebuildXLSXBlock 把译文写回一个 <si>/<is> 块
func rebuildXLSXBlock(b xlsxBlock, translated string) string {
	if !b.Translatable || strings.TrimSpace(translated) == "" || len(b.Runs) == 0 {
		return b.Content
	}

	if countTextRuns(b.Runs) > 1 {
		if runTexts, valid := DecodeRunMarks(translated, len(b.Runs)); valid {
			for j := range runTexts {
				runTexts[j] = SanitizeSegmentText(runTexts[j])
			}
			return replaceBlockRuns(b, runTexts)
		}
		// 回退:整段译文放第一个有文字的 run,其余清空
		plain := SanitizeSegmentText(StripRunMarks(translated))
		return replaceBlockFirstRun(b, plain)
	}

	plain := SanitizeSegmentText(StripRunMarks(translated))
	return replaceBlockFirstRun(b, plain)
}

func replaceBlockRuns(b xlsxBlock, runTexts []string) string {
	result := b.Content
	for i := len(b.Runs) - 1; i >= 0; i-- {
		run := b.Runs[i]
		text := ""
		if i < len(runTexts) {
			text = runTexts[i]
		}
		result = result[:run.StartInPara] + run.OpenTag + encodeXMLEntities(text) + run.CloseTag + result[run.EndInPara:]
	}
	return result
}

func replaceBlockFirstRun(b xlsxBlock, translated string) string {
	firstIdx := -1
	for i, run := range b.Runs {
		if strings.TrimSpace(run.Text) != "" {
			firstIdx = i
			break
		}
	}
	if firstIdx == -1 {
		firstIdx = 0
	}
	result := b.Content
	for i := len(b.Runs) - 1; i >= 0; i-- {
		run := b.Runs[i]
		text := ""
		if i == firstIdx {
			text = translated
		}
		result = result[:run.StartInPara] + run.OpenTag + encodeXMLEntities(text) + run.CloseTag + result[run.EndInPara:]
	}
	return result
}

// rebuildZipWithFiles 复制源 zip,替换指定文件
func rebuildZipWithFiles(sourcePath string, replaced map[string]string, outputPath string) error {
	srcReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return fmt.Errorf("open source xlsx: %w", err)
	}
	defer srcReader.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer outFile.Close()

	zw := zip.NewWriter(outFile)
	defer zw.Close()

	for _, f := range srcReader.File {
		if newContent, ok := replaced[f.Name]; ok {
			if err := addBytesToZip(zw, f.Name, []byte(newContent)); err != nil {
				return fmt.Errorf("write %q: %w", f.Name, err)
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry %q: %w", f.Name, err)
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return fmt.Errorf("read zip entry %q: %w", f.Name, err)
		}
		if err := addBytesToZip(zw, f.Name, data); err != nil {
			return fmt.Errorf("write zip entry %q: %w", f.Name, err)
		}
	}
	return nil
}

// ─────────────────────────────────────────────
// 旧接口兼容(引擎已改用 Segment 协议,此处仅兜底)
// ─────────────────────────────────────────────

func (p *XLSXProcessor) Extract(filePath string) (string, error) {
	segs, err := p.ExtractSegments()
	if err != nil {
		return "", err
	}
	lines := make([]string, len(segs))
	for i, s := range segs {
		lines[i] = s.Text
	}
	return strings.Join(lines, "\n"), nil
}

func (p *XLSXProcessor) Rebuild(translatedText string, outputPath string) error {
	segs, err := p.ExtractSegments()
	if err != nil {
		return err
	}
	lines := strings.Split(translatedText, "\n")
	m := map[string]string{}
	for i, s := range segs {
		if i < len(lines) {
			m[s.ID] = lines[i]
		}
	}
	return p.RebuildSegments(m, outputPath)
}
