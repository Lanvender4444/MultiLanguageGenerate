package filetype

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"path/filepath"
	"strings"
)

type FileType int

const (
	FileTypeUnknown FileType = iota
	FileTypeMarkdown
	FileTypeHTML
	FileTypeDOCX
	FileTypeXLSX
	FileTypePPTX
	FileTypeEPUB
	FileTypePlainText
	FileTypeCSV
	FileTypeJSON
	FileTypeXML
	FileTypeSRT
	FileTypePO
	FileTypeOldDoc
	FileTypeOldXls
)

type TypeInfo struct {
	Type        FileType
	Ext         string
	Description string
}

func Detect(filePath string, head []byte) FileType {
	// 1) ZIP magic: PK\x03\x04
	if len(head) >= 4 && binary.LittleEndian.Uint32(head[:4]) == 0x04034b50 {
		return detectZipType(filePath)
	}
	// 2) OLE2 (old Office): D0CF11E0A1B11AE1
	if len(head) >= 8 && bytes.Equal(head[:8], []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}) {
		return detectCompoundType(filePath)
	}
	// 3) Text
	return detectTextType(filePath, head)
}

func detectZipType(filePath string) FileType {
	r, err := zip.OpenReader(filePath)
	if err != nil {
		return detectByExtension(filePath)
	}
	defer r.Close()

	if len(r.File) == 0 {
		return FileTypeUnknown
	}

	first := r.File[0].Name
	if first == "mimetype" {
		rc, err := r.File[0].Open()
		if err == nil {
			buf := make([]byte, 64)
			n, _ := io.ReadFull(rc, buf)
			rc.Close()
			if n >= 20 && strings.Contains(string(buf[:n]), "application/epub+zip") {
				return FileTypeEPUB
			}
		}
	}

	for _, f := range r.File {
		switch {
		case f.Name == "word/document.xml":
			return FileTypeDOCX
		case f.Name == "xl/workbook.xml":
			return FileTypeXLSX
		case strings.HasPrefix(f.Name, "xl/"):
			return FileTypeXLSX
		case strings.HasPrefix(f.Name, "ppt/"):
			return FileTypePPTX
		}
	}

	if strings.HasSuffix(strings.ToLower(filePath), ".epub") {
		return FileTypeEPUB
	}
	return FileTypeUnknown
}

func detectCompoundType(filePath string) FileType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".doc":
		return FileTypeOldDoc
	case ".xls":
		return FileTypeOldXls
	}
	return FileTypeOldDoc
}

func detectTextType(filePath string, head []byte) FileType {
	text := string(head)
	if len(text) >= 3 && text[:3] == "\xEF\xBB\xBF" {
		text = text[3:]
	}
	text = strings.TrimLeft(text, " \t\r\n")
	if len(text) == 0 {
		return detectByExtension(filePath)
	}

	if (text[0] == '{' && text[len(text)-1] == '}') || (text[0] == '[' && text[len(text)-1] == ']') {
		return FileTypeJSON
	}

	check := text[:min(len(text), 200)]
	if strings.Contains(check, "<html") || strings.Contains(check, "<!DOCTYPE html") {
		return FileTypeHTML
	}
	if strings.HasPrefix(text, "<?xml") || strings.Contains(check, "</") {
		return FileTypeXML
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".md", ".markdown", ".rst":
		return FileTypeMarkdown
	case ".html", ".htm":
		return FileTypeHTML
	case ".csv", ".tsv":
		return FileTypeCSV
	case ".srt", ".ass", ".ssa", ".vtt":
		return FileTypeSRT
	case ".po", ".pot":
		return FileTypePO
	case ".json":
		return FileTypeJSON
	case ".xml":
		return FileTypeXML
	}
	return FileTypePlainText
}

func detectByExtension(filePath string) FileType {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".md", ".markdown", ".rst":
		return FileTypeMarkdown
	case ".html", ".htm":
		return FileTypeHTML
	case ".docx":
		return FileTypeDOCX
	case ".xlsx":
		return FileTypeXLSX
	case ".pptx":
		return FileTypePPTX
	case ".epub":
		return FileTypeEPUB
	case ".csv", ".tsv":
		return FileTypeCSV
	case ".json":
		return FileTypeJSON
	case ".xml":
		return FileTypeXML
	case ".srt", ".ass", ".ssa", ".vtt":
		return FileTypeSRT
	case ".po", ".pot":
		return FileTypePO
	case ".doc":
		return FileTypeOldDoc
	case ".xls":
		return FileTypeOldXls
	case ".txt", ".text", ".log":
		return FileTypePlainText
	}
	return FileTypePlainText
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TypeInfoOf(ft FileType) TypeInfo {
	switch ft {
	case FileTypeMarkdown:
		return TypeInfo{Type: ft, Ext: ".md", Description: "Markdown"}
	case FileTypeHTML:
		return TypeInfo{Type: ft, Ext: ".html", Description: "HTML"}
	case FileTypeDOCX:
		return TypeInfo{Type: ft, Ext: ".docx", Description: "Word Document"}
	case FileTypeXLSX:
		return TypeInfo{Type: ft, Ext: ".xlsx", Description: "Excel Workbook"}
	case FileTypePPTX:
		return TypeInfo{Type: ft, Ext: ".pptx", Description: "PowerPoint"}
	case FileTypeEPUB:
		return TypeInfo{Type: ft, Ext: ".epub", Description: "EPUB eBook"}
	case FileTypePlainText:
		return TypeInfo{Type: ft, Ext: ".txt", Description: "Plain Text"}
	case FileTypeCSV:
		return TypeInfo{Type: ft, Ext: ".csv", Description: "CSV"}
	case FileTypeJSON:
		return TypeInfo{Type: ft, Ext: ".json", Description: "JSON"}
	case FileTypeXML:
		return TypeInfo{Type: ft, Ext: ".xml", Description: "XML"}
	case FileTypeSRT:
		return TypeInfo{Type: ft, Ext: ".srt", Description: "Subtitle"}
	case FileTypePO:
		return TypeInfo{Type: ft, Ext: ".po", Description: "Gettext PO"}
	case FileTypeOldDoc:
		return TypeInfo{Type: ft, Ext: ".doc", Description: "Legacy Word"}
	case FileTypeOldXls:
		return TypeInfo{Type: ft, Ext: ".xls", Description: "Legacy Excel"}
	default:
		return TypeInfo{Type: ft, Ext: "", Description: "Unknown"}
	}
}
