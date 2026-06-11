package processor

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"math"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

// ─────────────────────────────────────────────
// 数据结构
// ─────────────────────────────────────────────

// RunInfo 描述一个 <w:t> 节点在 paraContent 中的位置与内容
type RunInfo struct {
	OpenTag     string // 完整的开标签，例如 <w:t> 或 <w:t xml:space="preserve">
	Text        string // 解码后的纯文字内容
	StartInPara int    // 在 paraContent 中的起始字节偏移（含 <w:t>）
	EndInPara   int    // 在 paraContent 中的结束字节偏移（含 </w:t>）
}

// ParagraphInfo 描述一个完整的 <w:p>...</w:p> 块
type ParagraphInfo struct {
	OpenTag      string    // <w:p> 或带属性的 <w:p ...>
	ParaContent  string    // <w:p> 和 </w:p> 之间的原始 XML
	Runs         []RunInfo // 该段落内所有 <w:t> 节点
	FullText     string    // 所有 run 拼接的完整文字（发给 LLM 翻译用）
	HasText      bool      // 该段落是否含有可翻译的文字
	StartInXML   int       // 整个 <w:p...> 在原始 XML 中的起始字节偏移
	EndInXML     int       // </w:p> 结束位置（不含）在原始 XML 中的字节偏移
}

// ─────────────────────────────────────────────
// 正则表达式（编译一次，全局复用）
// ─────────────────────────────────────────────

var (
	// 匹配 <w:p> 或 <w:p ...>（不跨行，属性里不含 >）
	paraOpenRe = regexp.MustCompile(`<w:p(?:\s[^>]*)?>`)

	// 匹配完整的 <w:t ...>...</w:t>，捕获组1=开标签内容，捕获组2=文字内容
	wtTagRe = regexp.MustCompile(`(<w:t(?:[^>]*)>)(.*?)</w:t>`)
)

// ─────────────────────────────────────────────
// DOCXProcessor
// ─────────────────────────────────────────────

type DOCXProcessor struct {
	SourcePath string

	// 内部缓存：Parse 一次，Extract 和 Rebuild 共用
	cachedXML        string
	cachedParagraphs []ParagraphInfo
}

func (p *DOCXProcessor) FileType() filetype.FileType {
	return filetype.FileTypeDOCX
}

// ─────────────────────────────────────────────
// Extract：提取可翻译的段落文字，每行一段
// ─────────────────────────────────────────────

func (p *DOCXProcessor) Extract(filePath string) (string, error) {
	xmlContent, err := readDocumentXML(filePath)
	if err != nil {
		return "", err
	}

	paragraphs := parseParagraphs(xmlContent)

	// 缓存供 Rebuild 使用（仅当 filePath == SourcePath 时有效）
	if filePath == p.SourcePath {
		p.cachedXML = xmlContent
		p.cachedParagraphs = paragraphs
	}

	var lines []string
	for _, para := range paragraphs {
		if para.HasText {
			lines = append(lines, para.FullText)
		}
	}
	return strings.Join(lines, "\n"), nil
}

// ─────────────────────────────────────────────
// Rebuild：将翻译后的文字写回 docx，保留所有格式
// ─────────────────────────────────────────────

func (p *DOCXProcessor) Rebuild(translatedText string, outputPath string) error {
	// 优先使用缓存，避免重复读取和解析
	xmlContent := p.cachedXML
	paragraphs := p.cachedParagraphs

	if xmlContent == "" || paragraphs == nil {
		var err error
		xmlContent, err = readDocumentXML(p.SourcePath)
		if err != nil {
			return err
		}
		paragraphs = parseParagraphs(xmlContent)
	}

	// 译文按 \n 切分，与 Extract 输出的行数严格对应
	translatedLines := strings.Split(translatedText, "\n")

	// 只有 HasText 的段落消耗译文索引，其余段落原样保留
	transIdx := 0
	paraContents := make([]string, len(paragraphs))
	for i, para := range paragraphs {
		if para.HasText && transIdx < len(translatedLines) {
			paraContents[i] = rebuildParagraphContent(para, translatedLines[transIdx])
			transIdx++
		} else {
			paraContents[i] = para.ParaContent
		}
	}

	rebuiltXML := rebuildXML(xmlContent, paragraphs, paraContents)
	return rebuildDocx(p.SourcePath, rebuiltXML, outputPath)
}

// ─────────────────────────────────────────────
// parseParagraphs：核心解析，一次扫描建立完整索引
// ─────────────────────────────────────────────

func parseParagraphs(xmlContent string) []ParagraphInfo {
	var result []ParagraphInfo
	pos := 0

	for pos < len(xmlContent) {
		loc := paraOpenRe.FindStringIndex(xmlContent[pos:])
		if loc == nil {
			break
		}

		absParaStart := pos + loc[0]
		openTag := xmlContent[absParaStart : pos+loc[1]]
		contentStart := pos + loc[1]

		closeIdx := strings.Index(xmlContent[contentStart:], "</w:p>")
		if closeIdx == -1 {
			break
		}

		paraContent := xmlContent[contentStart : contentStart+closeIdx]
		absParaEnd := contentStart + closeIdx + len("</w:p>")

		// 提取该段落内所有 <w:t> run
		runMatches := wtTagRe.FindAllStringSubmatchIndex(paraContent, -1)
		var runs []RunInfo
		var sb strings.Builder

		for _, m := range runMatches {
			// m[0]:m[1] = 整个匹配
			// m[2]:m[3] = 捕获组1：开标签 <w:t ...>
			// m[4]:m[5] = 捕获组2：文字内容
			openTagStr := paraContent[m[2]:m[3]]
			rawText := paraContent[m[4]:m[5]]
			decodedText := decodeXMLEntities(rawText)

			runs = append(runs, RunInfo{
				OpenTag:     openTagStr,
				Text:        decodedText,
				StartInPara: m[0],
				EndInPara:   m[1],
			})
			sb.WriteString(decodedText)
		}

		fullText := sb.String()
		result = append(result, ParagraphInfo{
			OpenTag:     openTag,
			ParaContent: paraContent,
			Runs:        runs,
			FullText:    fullText,
			HasText:     len(runs) > 0 && len(strings.TrimSpace(fullText)) > 0,
			StartInXML:  absParaStart,
			EndInXML:    absParaEnd,
		})

		pos = absParaEnd
	}

	return result
}

// ─────────────────────────────────────────────
// rebuildParagraphContent：按 run 比例分配译文，从后往前替换
// rebuildParagraphContent2：放进第一个run
// ─────────────────────────────────────────────

func rebuildParagraphContent(para ParagraphInfo, translatedFull string) string {
	if len(para.Runs) == 0 {
		return para.ParaContent
	}

	// 如果只有一个 run，直接替换，无需比例计算
	if len(para.Runs) == 1 {
		run := para.Runs[0]
		newTag := run.OpenTag + encodeXMLEntities(translatedFull) + "</w:t>"
		return para.ParaContent[:run.StartInPara] + newTag + para.ParaContent[run.EndInPara:]
	}

	// 多个 run：按原文字符比例分配译文
	origRunes := []rune(para.FullText)
	transRunes := []rune(translatedFull)
	totalOrig := len(origRunes)
	totalTrans := len(transRunes)

	runTexts := make([]string, len(para.Runs))
	cursor := 0

	for i, run := range para.Runs {
		if i == len(para.Runs)-1 {
			// 最后一个 run 拿全部剩余，避免因四舍五入丢字
			runTexts[i] = string(transRunes[cursor:])
		} else {
			runOrigLen := len([]rune(run.Text))
			var allocated int
			if totalOrig == 0 {
				allocated = 0
			} else {
				allocated = int(math.Round(float64(runOrigLen) / float64(totalOrig) * float64(totalTrans)))
			}
			end := cursor + allocated
			if end > totalTrans {
				end = totalTrans
			}
			runTexts[i] = string(transRunes[cursor:end])
			cursor = end
		}
	}

	// 从后往前替换，保证前面 run 的字节偏移不受后面替换影响
	result := para.ParaContent
	for i := len(para.Runs) - 1; i >= 0; i-- {
		run := para.Runs[i]
		newTag := run.OpenTag + encodeXMLEntities(runTexts[i]) + "</w:t>"
		result = result[:run.StartInPara] + newTag + result[run.EndInPara:]
	}

	return result
}

func rebuildParagraphContent2(para ParagraphInfo, translatedFull string) string {
    if len(para.Runs) == 0 {
        return para.ParaContent
    }

    // 找第一个有实际文字内容的 run，译文全部放进它
    // 其余 run 的 <w:t> 内容清空（保留 run 的格式属性）
    firstIdx := -1
    for i, run := range para.Runs {
        if strings.TrimSpace(run.Text) != "" {
            firstIdx = i
            break
        }
    }
    if firstIdx == -1 {
        // 全是空 run，原样返回
        return para.ParaContent
    }

    // 从后往前替换，保证字节偏移不偏移
    result := para.ParaContent
    for i := len(para.Runs) - 1; i >= 0; i-- {
        run := para.Runs[i]
        var newContent string
        if i == firstIdx {
            // 第一个有文字的 run：放入全部译文
            newContent = run.OpenTag + encodeXMLEntities(translatedFull) + "</w:t>"
        } else {
            // 其余 run：清空文字，保留格式标签结构
            newContent = run.OpenTag + "</w:t>"
        }
        result = result[:run.StartInPara] + newContent + result[run.EndInPara:]
    }
    return result
}

// ─────────────────────────────────────────────
// rebuildXML：将各段落的新 paraContent 写回原始 XML
// ─────────────────────────────────────────────

func rebuildXML(originalXML string, paragraphs []ParagraphInfo, paraContents []string) string {
	if len(paragraphs) == 0 {
		return originalXML
	}

	var sb strings.Builder
	sb.Grow(len(originalXML))

	pos := 0
	for i, para := range paragraphs {
		// 写入本段落之前的原始内容（包括上一个 </w:p> 到这个 <w:p> 之间的部分）
		sb.WriteString(originalXML[pos:para.StartInXML])

		// 写入重建后的段落
		sb.WriteString(para.OpenTag)
		sb.WriteString(paraContents[i])
		sb.WriteString("</w:p>")

		pos = para.EndInXML
	}

	// 写入最后一个段落之后的剩余 XML
	sb.WriteString(originalXML[pos:])

	return sb.String()
}

// ─────────────────────────────────────────────
// ZIP / 文件操作
// ─────────────────────────────────────────────

func readDocumentXML(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx zip %q: %w", filePath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open document.xml: %w", err)
			}
			data, err := io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("read document.xml: %w", err)
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("word/document.xml not found in %q", filePath)
}

func rebuildDocx(sourcePath string, newDocumentXML string, outputPath string) error {
	srcReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return fmt.Errorf("open source docx: %w", err)
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
		if f.Name == "word/document.xml" {
			if err := addBytesToZip(zw, f.Name, []byte(newDocumentXML)); err != nil {
				return fmt.Errorf("write document.xml: %w", err)
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

func addBytesToZip(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}

// ─────────────────────────────────────────────
// XML 实体编解码
// ─────────────────────────────────────────────

func decodeXMLEntities(s string) string {
	// 顺序很重要：&amp; 必须最后 decode，最先 encode
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}

func encodeXMLEntities(s string) string {
	// &amp; 必须最先 encode，否则会二次转义其他实体
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}