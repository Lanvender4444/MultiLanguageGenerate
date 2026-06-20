package glossary

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/processor"
)

// Target 一个目标语言（代码 + 显示名）
type Target struct {
	Code string
	Name string
}

// GenerateInput AI 生成词表的输入
type GenerateInput struct {
	// Prompt 用户提示词，描述要提取/构造哪些专业名词、风格倾向等
	Prompt string
	// SourceLanguage 源语言显示名
	SourceLanguage string
	// Targets 目标语言列表
	Targets []Target
	// Context 由 RAG 从知识库检索出的参考片段（专业例子/风格）
	Context []string
	// Model 模型名
	Model string
}

// Generate 调用 LLM 依据提示词 + 知识库参考，生成专业名词词表。
func Generate(ctx context.Context, provider llm.Provider, in GenerateInput) (*Glossary, error) {
	if provider == nil {
		return nil, fmt.Errorf("glossary: provider is nil")
	}
	if strings.TrimSpace(in.Prompt) == "" {
		return nil, fmt.Errorf("glossary: prompt is required")
	}

	sysPrompt := buildGenSystemPrompt(in)
	userMsg := buildGenUserMessage(in)

	raw, err := provider.Translate(ctx, llm.TranslateRequest{
		SourceText:     userMsg,
		SourceLanguage: in.SourceLanguage,
		TargetLanguage: "professional terminology glossary (JSON)",
		TargetCode:     "glossary",
		Model:          in.Model,
		SourceType:     filetype.FileTypePlainText,
		SystemPrompt:   sysPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("glossary: LLM call failed: %w", err)
	}

	cleaned := processor.ExtractJSONObject(processor.StripCodeFence(raw))
	if cleaned == "" {
		return nil, fmt.Errorf("glossary: no JSON object found in model output")
	}

	var parsed struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Entries     []Entry `json:"entries"`
	}
	if err := json.Unmarshal([]byte(cleaned), &parsed); err != nil {
		return nil, fmt.Errorf("glossary: parse model JSON: %w", err)
	}

	g := &Glossary{Name: parsed.Name, Description: parsed.Description, Entries: parsed.Entries}
	// 清洗：去掉空 term 的条目，确保 translation map 非 nil
	clean := make([]Entry, 0, len(g.Entries))
	for _, e := range g.Entries {
		if strings.TrimSpace(e.Term) == "" {
			continue
		}
		if e.Translation == nil {
			e.Translation = map[string]string{}
		}
		clean = append(clean, e)
	}
	g.Entries = clean
	return g, nil
}

func buildGenSystemPrompt(in GenerateInput) string {
	var codes []string
	for _, t := range in.Targets {
		codes = append(codes, fmt.Sprintf("%q (%s)", t.Code, t.Name))
	}
	return fmt.Sprintf(`You are a professional terminology and localization expert.
Your job: from the user's instruction and the provided reference materials, build a TERMINOLOGY GLOSSARY for translation from %s into the following target languages: %s.

Identify domain-specific terms, proper nouns, character/person names, product/brand names, and any expressions that need a fixed, consistent, context-aware translation. For each, provide the preferred translation in EVERY target language listed above.

Output REQUIREMENTS (strict):
1. Reply with a SINGLE JSON object ONLY. No code fences, no commentary, nothing before or after.
2. JSON schema:
{
  "name": "short glossary name",
  "description": "one-line description",
  "entries": [
    {
      "term": "source-language term",
      "type": "person|place|tech|brand|term|other",
      "context": "when this translation applies / disambiguation note",
      "aliases": ["other spellings or forms"],
      "translation": { %s }
    }
  ]
}
3. "translation" keys MUST be exactly the target language codes listed above; values are the chosen translations.
4. Base your choices on the reference materials when relevant (preserve their style and preferred renderings).
5. Do NOT invent terms unrelated to the instruction. Prefer precision over quantity.
6. Ensure the JSON is strictly valid (escape quotes/backslashes).`,
		fallback(in.SourceLanguage, "the source language"),
		strings.Join(codes, ", "),
		exampleTranslationKeys(in.Targets))
}

func buildGenUserMessage(in GenerateInput) string {
	var sb strings.Builder
	sb.WriteString("INSTRUCTION:\n")
	sb.WriteString(in.Prompt)
	sb.WriteString("\n")
	if len(in.Context) > 0 {
		sb.WriteString("\nREFERENCE MATERIALS (from the professional knowledge base; use them to guide terminology and style):\n")
		for i, c := range in.Context {
			fmt.Fprintf(&sb, "\n[REF %d]\n%s\n", i+1, c)
		}
	}
	return sb.String()
}

// exampleTranslationKeys 生成 schema 里 translation 字段的示例键
func exampleTranslationKeys(targets []Target) string {
	parts := make([]string, 0, len(targets))
	for _, t := range targets {
		parts = append(parts, fmt.Sprintf("%q: \"...\"", t.Code))
	}
	return strings.Join(parts, ", ")
}

func fallback(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
