package processor

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type DOCXProcessor struct {
	SourcePath string
}

func (p *DOCXProcessor) Extract(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", fmt.Errorf("open docx: %w", err)
	}
	defer r.Close()

	var xmlDoc []byte
	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", fmt.Errorf("open document.xml: %w", err)
			}
			xmlDoc, err = io.ReadAll(rc)
			rc.Close()
			if err != nil {
				return "", fmt.Errorf("read document.xml: %w", err)
			}
			break
		}
	}

	if len(xmlDoc) == 0 {
		return "", fmt.Errorf("word/document.xml not found in docx")
	}

	paragraphs := extractParagraphs(string(xmlDoc))
	var sb strings.Builder
	for i, para := range paragraphs {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(para)
	}
	return sb.String(), nil
}

func (p *DOCXProcessor) Rebuild(translatedText string, outputPath string) error {
	originalXML, err := p.readDocumentXML(p.SourcePath)
	if err != nil {
		return err
	}

	translatedParagraphs := strings.Split(translatedText, "\n")
	rebuiltXML, err := replaceParagraphsInXML(originalXML, translatedParagraphs)
	if err != nil {
		return err
	}

	return p.rebuildDocx(p.SourcePath, rebuiltXML, outputPath)
}

func (p *DOCXProcessor) FileType() filetype.FileType {
	return filetype.FileTypeDOCX
}

var wtTagRe = regexp.MustCompile(`<w:t[^>]*>(.*?)</w:t>`)

func extractParagraphs(xmlContent string) []string {
	paraRe := regexp.MustCompile(`<w:p[ >]`)
	parts := paraRe.Split(xmlContent, -1)

	var paragraphs []string
	for i := 1; i < len(parts); i++ {
		endIdx := strings.Index(parts[i], "</w:p>")
		if endIdx == -1 {
			continue
		}
		paraContent := parts[i][:endIdx]

		matches := wtTagRe.FindAllStringSubmatch(paraContent, -1)
		var texts []string
		for _, m := range matches {
			if len(m) > 1 {
				decoded := decodeXMLEntities(m[1])
				texts = append(texts, decoded)
			}
		}
		if len(texts) > 0 {
			paragraphs = append(paragraphs, strings.Join(texts, ""))
		}
	}
	return paragraphs
}

func replaceParagraphsInXML(originalXML string, translatedParagraphs []string) (string, error) {
	paraRe := regexp.MustCompile(`<w:p[ >]`)
	parts := paraRe.Split(originalXML, -1)

	paraIdx := 0
	var result strings.Builder
	result.WriteString(parts[0])

	for i := 1; i < len(parts); i++ {
		endIdx := strings.Index(parts[i], "</w:p>")
		if endIdx == -1 {
			result.WriteString("<w:p")
			result.WriteString(parts[i])
			continue
		}

		paraContent := parts[i][:endIdx]
		rest := parts[i][endIdx+len("</w:p>"):]

		matches := wtTagRe.FindAllStringSubmatchIndex(paraContent, -1)

		if len(matches) == 0 {
			result.WriteString("<w:p")
			result.WriteString(parts[i])
			continue
		}

		if paraIdx < len(translatedParagraphs) {
			newText := encodeXMLEntities(translatedParagraphs[paraIdx])
			modified := replaceWTTexts(paraContent, newText)
			result.WriteString("<w:p")
			result.WriteString(modified)
			result.WriteString("</w:p>")
			result.WriteString(rest)
			paraIdx++
		} else {
			result.WriteString("<w:p")
			result.WriteString(parts[i])
		}
	}

	return result.String(), nil
}

func replaceWTTexts(paraContent string, newText string) string {
	matches := wtTagRe.FindAllStringSubmatchIndex(paraContent, -1)
	if len(matches) == 0 {
		return paraContent
	}

	firstStart := matches[0][0]
	lastEnd := matches[len(matches)-1][1]

	spaceAttr := ""
	if strings.Contains(paraContent[matches[0][0]:matches[0][1]], `xml:space="preserve"`) {
		spaceAttr = ` xml:space="preserve"`
	}

	prefix := paraContent[:firstStart]
	suffix := paraContent[lastEnd:]

	return prefix + fmt.Sprintf(`<w:t%s>%s</w:t>`, spaceAttr, newText) + suffix
}

func decodeXMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", "\"")
	s = strings.ReplaceAll(s, "&apos;", "'")
	return s
}

func encodeXMLEntities(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func (p *DOCXProcessor) readDocumentXML(filePath string) (string, error) {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return "", err
	}
	defer r.Close()

	for _, f := range r.File {
		if f.Name == "word/document.xml" {
			rc, err := f.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()
			data, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
	}
	return "", fmt.Errorf("word/document.xml not found")
}

func (p *DOCXProcessor) rebuildDocx(sourcePath string, newDocumentXML string, outputPath string) error {
	srcReader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return err
	}
	defer srcReader.Close()

	outFile, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for _, f := range srcReader.File {
		if f.Name == "word/document.xml" {
			if err := p.addFileToZip(zipWriter, f.Name, []byte(newDocumentXML)); err != nil {
				return err
			}
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return err
		}

		if err := p.addFileToZip(zipWriter, f.Name, data); err != nil {
			return err
		}
	}

	return nil
}

func (p *DOCXProcessor) addFileToZip(zw *zip.Writer, name string, data []byte) error {
	w, err := zw.Create(name)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	return err
}