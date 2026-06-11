package processor

import (
	"fmt"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

type Processor interface {
	Extract(filePath string) (string, error)
	Rebuild(translatedText string, outputPath string) error
	FileType() filetype.FileType
}

func NewProcessor(ft filetype.FileType, sourceFilePath string) Processor {
	switch ft {
	case filetype.FileTypeDOCX:
		return &DOCXProcessor{SourcePath: sourceFilePath}
	case filetype.FileTypeHTML:
		return &HTMLProcessor{SourcePath: sourceFilePath}
	case filetype.FileTypeMarkdown, filetype.FileTypeSRT, filetype.FileTypePO,
		filetype.FileTypeCSV, filetype.FileTypeJSON, filetype.FileTypeXML,
		filetype.FileTypePlainText, filetype.FileTypeEPUB,
		filetype.FileTypeXLSX, filetype.FileTypePPTX,
		filetype.FileTypeOldDoc, filetype.FileTypeOldXls,
		filetype.FileTypeUnknown:
		return &TextProcessor{SourcePath: sourceFilePath, FT: ft}
	default:
		return &TextProcessor{SourcePath: sourceFilePath, FT: ft}
	}
}

func BuildSystemPrompt(srcLang, dstLang, dstCode string, ft filetype.FileType) string {
	base := fmt.Sprintf("You are a professional technical document translator.\nTranslate the following document from %s to %s (%s).\n", srcLang, dstLang, dstCode)

	switch ft {
	case filetype.FileTypeMarkdown:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve ALL Markdown formatting EXACTLY: headings (#, ##, ###), bold (**text**), italic (*text*), links ([text](url)), images (![alt](url)), code blocks (` + "```" + `), inline code (` + "`" + `), blockquotes (>), lists (-, *, 1.), horizontal rules (---), tables (| col | col |)
2. Do NOT translate content inside code blocks or inline code
3. Do NOT translate URLs, file paths, variable names, or command-line arguments
4. Do NOT translate HTML tags in Markdown - keep them exactly as-is
5. Preserve the exact line structure: same number of blank lines, same line breaks
6. Preserve ALL table formatting: the | delimiters, alignment (:---, :---:, ---:), and header separator rows MUST remain untouched. Only translate the CELL CONTENT between | characters
7. Preserve all footnote references ([^1]), reference-style links ([1]), and their definitions
8. Output ONLY the translated content, no explanations or preamble`

	case filetype.FileTypeHTML:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve ALL HTML tags, attributes, and structure EXACTLY as-is. Do NOT add, remove, or modify any tags or attributes
2. Only translate TEXT CONTENT between > and < characters
3. Do NOT translate content inside <code>, <pre>, <script>, <style> tags
4. Do NOT translate attribute values like href, src, alt (unless alt is visible text)
5. Preserve class names, IDs, data- attributes exactly as-is
6. Keep the exact whitespace and indentation structure
7. Preserve HTML entities (&amp;, &lt;, &gt;, etc.) - do NOT convert them
8. Output ONLY the translated HTML, no explanations`

	case filetype.FileTypeCSV:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve the CSV structure EXACTLY: same number of columns, same delimiters (commas, tabs, etc.)
2. Preserve all quoted fields and their quoting style
3. Only translate the CELL CONTENT, not the structure
4. If a cell contains a formula or numeric value, leave it unchanged
5. Preserve header rows - translate header labels only
6. Do NOT add or remove any rows or columns
7. Output ONLY the translated CSV, no explanations`

	case filetype.FileTypeJSON:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve the JSON structure EXACTLY: same keys, same nesting, same data types
2. Only translate STRING VALUES, NOT the keys
3. Do NOT translate numeric values, booleans, or null
4. Preserve all escaping (\n, \t, \", \\, etc.) in string values
5. Keep the same indentation/formatting as the original
6. Output ONLY valid JSON, no explanations`

	case filetype.FileTypeXML:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve ALL XML tags, attributes, namespace declarations EXACTLY as-is
2. Only translate TEXT CONTENT between > and < characters
3. Do NOT translate attribute values unless they are human-visible text
4. Preserve CDATA sections - translate only the text inside, keep CDATA markers
5. Preserve XML processing instructions and comments
6. Keep the exact whitespace and indentation structure
7. Output ONLY the translated XML, no explanations`

	case filetype.FileTypeSRT:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve ALL subtitle numbers EXACTLY as-is
2. Preserve ALL timestamps EXACTLY as-is (HH:MM:SS,mmm --> HH:MM:SS,mmm)
3. Only translate the TEXT LINES, never the numbers or timestamps
4. Preserve line breaks within subtitle blocks
5. Keep the exact same number of subtitle blocks
6. Do NOT merge or split subtitle blocks
7. Output ONLY the translated SRT content, no explanations`

	case filetype.FileTypePO:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Preserve ALL gettext PO structure: msgid, msgstr, msgid_plural, msgstr[0], msgstr[1], etc.
2. Only translate msgstr values, NEVER translate msgid values
3. Preserve all comments (#. #, #, #: etc.) exactly as-is
4. Preserve all format specifiers (%s, %d, {0}, etc.) in the translated text
5. Preserve all escape sequences (\n, \t, \\, \") exactly
6. Keep the exact same number of entries
7. Output ONLY the translated PO content, no explanations`

	case filetype.FileTypeDOCX, filetype.FileTypeXLSX:
		return base + `
STRICT RULES - you MUST follow these rules:
1. Translate ONLY the plain text content provided to you
2. Preserve ALL special markers like [BR], [TAB], [PAGE_BREAK] exactly as-is - these represent formatting in the original document
3. Do NOT add or remove any markers
4. Maintain paragraph structure - each line of input corresponds to one paragraph
5. If a paragraph is empty or contains only a marker, output it unchanged
6. CRITICAL: Do NOT output ANY Markdown syntax whatsoever. This content comes from a Word/Excel document.
   - Do NOT use **bold**, *italic*, __underline__ or any asterisk/underscore formatting
   - Do NOT use # headings, ## subheadings, or any hash-based formatting
   - Do NOT use | table | syntax, do NOT create Markdown tables
   - Do NOT use - * + bullet list markers at line starts
   - Do NOT use ` + "`" + `code` + "`" + ` or ` + "```" + `code blocks` + "```" + `
   - Do NOT use > blockquotes
   - Do NOT use [link](url) or ![image](url) syntax
   - Output ONLY plain text, exactly as structured in the input
7. Output ONLY the translated text, no explanations or preamble`

	default:
		return base + `
Rules:
1. Preserve ALL original formatting, structure, and whitespace
2. Do NOT translate content inside code blocks
3. Do NOT translate URLs, file paths, variable names
4. Keep the same line structure and whitespace as the original
5. Output ONLY the translated content, no explanations or preamble`
	}
}