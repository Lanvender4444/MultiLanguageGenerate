package translate

import (
	"context"
	"fmt"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/glossary"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/knowledge"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/language"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
)

// GlossaryEntry 一个专业名词条目（公共视图）
type GlossaryEntry struct {
	Term        string
	Type        string
	Context     string
	Aliases     []string
	Translation map[string]string
}

// GlossaryDoc 一份词表（公共视图）
type GlossaryDoc struct {
	Name        string
	Description string
	Entries     []GlossaryEntry
}

// GlossaryGenOptions 生成专业名词词表的参数。
//
// 工作流：从知识库按 Prompt 检索相关参考片段（RAG）→ 连同 Prompt 投喂给 AI
// → AI 产出包含各目标语言指定译法的词表 JSON。
type GlossaryGenOptions struct {
	// Prompt 用户提示词（必填），描述要构造哪些专业名词、风格倾向等。
	Prompt string
	// TargetCodes 目标语言代码（必填）。
	TargetCodes []string
	// SourceLanguage 源语言显示名（可选）。
	SourceLanguage string

	// LLM
	Provider string
	Model    string
	APIKey   string
	BaseURL  string

	// 知识库（RAG）
	KBDir      string   // 知识库目录；留空用默认
	SelectKB   []string // 勾选的文档名或标签；空=全部
	TopK       int      // 检索片段数；<=0 取 8
	EmbedModel string   // 配置后用向量检索，否则 BM25

	// OutputPath 非空时把生成的词表写入该 JSON 文件。
	OutputPath string
}

// GenerateGlossary 执行 RAG + AI 生成专业名词词表。
func GenerateGlossary(ctx context.Context, opts GlossaryGenOptions) (*GlossaryDoc, error) {
	if opts.Prompt == "" {
		return nil, fmt.Errorf("translate: glossary Prompt is required")
	}
	if len(opts.TargetCodes) == 0 {
		return nil, fmt.Errorf("translate: at least one target language code is required")
	}
	if opts.Provider == "" {
		return nil, fmt.Errorf("translate: Provider is required")
	}
	if _, ok := llm.GetProviderInfo(opts.Provider); !ok {
		return nil, fmt.Errorf("translate: unknown provider %q", opts.Provider)
	}
	if opts.APIKey == "" && opts.Provider != "ollama" {
		return nil, fmt.Errorf("translate: APIKey is required for provider %q", opts.Provider)
	}

	provider, err := llm.CreateProvider(opts.Provider, config.ProviderConfig{
		APIKey:  opts.APIKey,
		Model:   opts.Model,
		BaseURL: opts.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("translate: create provider: %w", err)
	}

	// ── RAG 检索参考片段 ──
	topK := opts.TopK
	if topK <= 0 {
		topK = 8
	}
	var contexts []string
	kb, kerr := knowledge.Open(opts.KBDir)
	if kerr == nil {
		embedder := makeEmbedder(opts.Provider, opts.APIKey, opts.BaseURL, opts.EmbedModel)
		if hits, herr := kb.Retrieve(ctx, opts.Prompt, topK, opts.SelectKB, embedder); herr == nil {
			for _, h := range hits {
				contexts = append(contexts, h.Text)
			}
		}
	}

	// ── 目标语言 ──
	targets := make([]glossary.Target, 0, len(opts.TargetCodes))
	for _, code := range opts.TargetCodes {
		targets = append(targets, glossary.Target{Code: code, Name: language.NameByCode(code)})
	}

	// ── AI 生成 ──
	g, gerr := glossary.Generate(ctx, provider, glossary.GenerateInput{
		Prompt:         opts.Prompt,
		SourceLanguage: opts.SourceLanguage,
		Targets:        targets,
		Context:        contexts,
		Model:          opts.Model,
	})
	if gerr != nil {
		return nil, gerr
	}

	if opts.OutputPath != "" {
		if err := g.Save(opts.OutputPath); err != nil {
			return nil, fmt.Errorf("translate: save glossary: %w", err)
		}
	}

	return toGlossaryDoc(g), nil
}

func toGlossaryDoc(g *glossary.Glossary) *GlossaryDoc {
	doc := &GlossaryDoc{Name: g.Name, Description: g.Description}
	for _, e := range g.Entries {
		doc.Entries = append(doc.Entries, GlossaryEntry{
			Term:        e.Term,
			Type:        e.Type,
			Context:     e.Context,
			Aliases:     e.Aliases,
			Translation: e.Translation,
		})
	}
	return doc
}
