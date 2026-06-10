package output

import (
	"path/filepath"
	"strings"
)

func BuildOutputPath(sourceFile, targetCode, outputDir string) string {
	ext := filepath.Ext(sourceFile)
	base := strings.TrimSuffix(filepath.Base(sourceFile), ext)
	outName := base + "_" + targetCode + ext

	if outputDir != "" {
		return filepath.Join(outputDir, outName)
	}
	return filepath.Join(filepath.Dir(sourceFile), outName)
}