package filetype

import (
	"os"
)

func ReadHead(path string, size int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, size)
	n, err := f.Read(buf)
	if err != nil {
		return nil, err
	}
	return buf[:n], nil
}

func DetectFile(path string) FileType {
	head, err := ReadHead(path, 8192)
	if err != nil {
		return detectByExtension(path)
	}
	return Detect(path, head)
}

func OutputExtForType(ft FileType, originalExt string) string {
	switch ft {
	case FileTypeDOCX:
		return ".docx"
	case FileTypeXLSX:
		return ".xlsx"
	case FileTypePPTX:
		return ".pptx"
	case FileTypeEPUB:
		return ".epub"
	case FileTypeHTML:
		return originalExt
	default:
		return originalExt
	}
}

func IsBinary(ft FileType) bool {
	switch ft {
	case FileTypeDOCX, FileTypeXLSX, FileTypePPTX, FileTypeEPUB, FileTypeOldDoc, FileTypeOldXls:
		return true
	default:
		return false
	}
}
