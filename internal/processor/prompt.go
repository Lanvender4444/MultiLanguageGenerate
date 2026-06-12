package processor

import (
	"fmt"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
)

// ─────────────────────────────────────────────
// Prompt 体系
//
// 结构:共享核心规则(所有类型生效) + 按文件类型的专项规则。
// DOCX/XLSX 走 Segment JSON 协议,使用 BuildSegmentSystemPrompt;
// 其余类型整文件翻译,使用 BuildSystemPrompt。
// ─────────────────────────────────────────────

// promptCore 所有翻译任务共享的核心约束
func promptCore(srcLang, dstLang, dstCode string) string {
	return fmt.Sprintf(`You are a professional document translator.
Task: translate from %s to %s (%s).

CORE RULES (apply to every task, highest priority):
1. NEVER modify the source file's format or structure. You translate text content ONLY; every structural character (tags, delimiters, markers, whitespace layout) must appear in your output exactly as in the input.
2. Output ONLY the translation result. No explanations, no preamble, no notes, and NEVER wrap the output in Markdown code fences.
3. Do NOT translate: URLs, email addresses, file paths, code identifiers, variable names, product/brand names that are conventionally kept in the source language.
4. Preserve ALL placeholders exactly: %%s, %%d, {0}, {name}, {{var}}, $VAR, <0>, etc.
5. Preserve escape sequences (\n, \t, \", \\) exactly as written.
6. If a piece of text is untranslatable (numbers, symbols, already in the target language), keep it unchanged.
7. Translate naturally and accurately; do not add or omit meaning.`, srcLang, dstLang, dstCode)
}

// BuildSystemPrompt 整文件翻译的系统提示(按文件类型附加专项规则)
func BuildSystemPrompt(srcLang, dstLang, dstCode string, ft filetype.FileType) string {
	base := promptCore(srcLang, dstLang, dstCode)

	switch ft {
	case filetype.FileTypeMarkdown:
		return base + `

FORMAT-SPECIFIC RULES (Markdown):
1. Preserve ALL Markdown syntax EXACTLY: headings (#), bold/italic (*, _), links [text](url), images ![alt](url), code fences, inline code, blockquotes (>), list markers (-, *, 1.), horizontal rules (---), tables (|).
2. Do NOT translate content inside code blocks or inline code.
3. Keep HTML tags embedded in Markdown unchanged.
4. Preserve the exact line structure: same number of lines, same blank lines.
5. In tables, translate ONLY cell content; never touch | delimiters or alignment rows (:---).
6. Preserve footnote refs ([^1]) and reference-style links.`

	case filetype.FileTypeHTML:
		return base + `

FORMAT-SPECIFIC RULES (HTML):
1. Preserve ALL tags, attributes, comments, and structure EXACTLY. Do NOT add, remove, reorder, or rewrite any tag or attribute.
2. Translate ONLY human-visible text: text nodes, plus title/alt/placeholder/aria-label attribute values.
3. Do NOT translate content inside <code>, <pre>, <script>, <style>.
4. Do NOT translate href, src, id, class, name, data-* attribute values.
5. Preserve HTML entities (&amp;, &lt;, &nbsp;, ...) exactly; do NOT convert them to characters.
6. Keep DOCTYPE, whitespace and indentation exactly as in the input.
7. The output must contain exactly the same sequence of tags as the input.`

	case filetype.FileTypeXML:
		return base + `

FORMAT-SPECIFIC RULES (XML):
1. Preserve ALL tags, attributes, namespaces, processing instructions, and comments EXACTLY.
2. Translate ONLY text content between tags.
3. Do NOT translate attribute values unless they are clearly human-visible text (e.g. label="...", title="...").
4. Preserve CDATA markers; translate only the text inside CDATA.
5. Preserve XML entities (&amp;, &lt;, ...) exactly.
6. The output MUST be well-formed XML with exactly the same element structure as the input.`

	case filetype.FileTypeJSON:
		return base + `

FORMAT-SPECIFIC RULES (JSON):
1. Output MUST be valid JSON parseable by a strict parser.
2. Preserve structure EXACTLY: same keys, same nesting, same order, same data types.
3. Translate ONLY string VALUES. NEVER translate keys.
4. Do NOT translate numbers, booleans, null, or string values that are identifiers/codes/URLs/paths.
5. Preserve all escaping inside strings (\n, \t, \", \\, \uXXXX).
6. Keep the original indentation style.`

	case filetype.FileTypeCSV:
		return base + `

FORMAT-SPECIFIC RULES (CSV):
1. Preserve the exact table shape: same number of rows and columns, same delimiter.
2. Preserve quoting style of every field; if a translated value contains the delimiter or a quote, wrap it in double quotes and escape inner quotes ("").
3. Translate ONLY cell text. Leave numbers, dates, formulas, IDs unchanged.
4. Never add, remove, merge, or reorder rows/columns.`

	case filetype.FileTypeSRT:
		return base + `

FORMAT-SPECIFIC RULES (Subtitles):
1. Preserve ALL sequence numbers and timestamps EXACTLY (HH:MM:SS,mmm --> HH:MM:SS,mmm).
2. Translate ONLY the subtitle text lines.
3. Keep the same number of blocks; never merge or split blocks.
4. Preserve line breaks within a block and blank lines between blocks.
5. Preserve styling tags like <i>, <b>, {\an8} exactly.`

	case filetype.FileTypePO:
		return base + `

FORMAT-SPECIFIC RULES (gettext PO):
1. Translate ONLY msgstr values. NEVER change msgid / msgid_plural.
2. Preserve all comments (#, #., #:, #,) and the header entry exactly.
3. Preserve format specifiers (%s, %d, {0}) and escape sequences in translations.
4. Keep the same number of entries and plural forms.`

	case filetype.FileTypeDOCX, filetype.FileTypeXLSX:
		// 兜底:正常流程下 DOCX/XLSX 应走 BuildSegmentSystemPrompt
		return base + `

FORMAT-SPECIFIC RULES (Office document, plain-text mode):
1. Each input line is one paragraph/cell; output exactly the same number of lines, translated line by line.
2. Output plain text ONLY - absolutely NO Markdown syntax (no **, #, |, backticks, list markers).
3. If a line is empty or untranslatable, output it unchanged.`

	default:
		return base + `

FORMAT-SPECIFIC RULES (plain text):
1. Keep the same line structure: same number of lines, same blank lines.
2. Preserve indentation and leading/trailing whitespace per line.`
	}
}

// BuildSegmentSystemPrompt Segment JSON 协议(DOCX/XLSX)的系统提示
func BuildSegmentSystemPrompt(srcLang, dstLang, dstCode string, ft filetype.FileType) string {
	doc := "a Word document"
	unit := "paragraph"
	if ft == filetype.FileTypeXLSX {
		doc = "an Excel workbook"
		unit = "cell"
	}

	return promptCore(srcLang, dstLang, dstCode) + fmt.Sprintf(`

INPUT/OUTPUT PROTOCOL (follow exactly):
The user message is a JSON object: {"id": "source text", ...}. Each entry is one %s extracted from %s.
You MUST reply with a single JSON object containing EXACTLY the same keys, where each value is the translation of the corresponding source text.

PROTOCOL RULES:
1. Reply with the JSON object ONLY: no code fences, no commentary, nothing before or after.
2. Every key from the input MUST appear in your output exactly once. Never add, drop, merge, or rename keys.
3. Translate each value independently; do not move content between entries.
4. Some values contain run markers like <r0>text</r0><r1>text</r1>. These mark formatting boundaries (bold, color, font changes) in the original %s:
   - Your translation MUST use the same markers: every <rN>...</rN> present in the source value must appear exactly once in the translated value, in the numeric order that reads naturally in the target language.
   - Distribute the translated text across the markers so each part carries the meaning of its source part. If a source part is empty (<r3></r3>), keep it empty.
   - Put NO text outside the markers.
   - Never invent new marker numbers.
5. Values WITHOUT run markers are plain text: output plain text with NO markup of any kind (no Markdown, no HTML, no markers).
6. If a value is untranslatable (numbers, codes, target-language text), return it unchanged.
7. Ensure the reply is strictly valid JSON: escape quotes and backslashes properly.`, unit, doc, unit)
}
