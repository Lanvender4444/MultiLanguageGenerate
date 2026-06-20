package translate

import (
	"context"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/knowledge"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
)

// KnowledgeDoc 知识库中的一个文档（公共视图）
type KnowledgeDoc struct {
	Name string
	Tags []string
}

// KnowledgeHit 一条知识库检索结果（公共视图）
type KnowledgeHit struct {
	Doc   string
	Tags  []string
	Text  string
	Score float64
}

// KBAddOptions 向知识库添加内容的参数
type KBAddOptions struct {
	Dir      string   // 知识库目录；留空用默认目录
	Path     string   // 待加入的源文件路径
	LinkMode string   // "copy"(默认) | "soft" | "hard"
	Tags     []string // 可选标签
}

// KBIndexOptions 重建知识库索引的参数（带可选 embedding）
type KBIndexOptions struct {
	Dir string // 知识库目录；留空用默认目录

	// 以下用于在建索引时计算向量（可选）。EmbedModel 为空则只建词法(BM25)索引。
	Provider   string
	APIKey     string
	BaseURL    string
	EmbedModel string
}

// KBSearchOptions 知识库检索参数
type KBSearchOptions struct {
	Dir      string
	Query    string
	TopK     int
	Selected []string // 限定文档名或标签；空=全部

	// 与 KBIndexOptions 相同，用于在向量索引存在时做语义检索。
	Provider   string
	APIKey     string
	BaseURL    string
	EmbedModel string
}

// KBAdd 把文件加入知识库（拷贝/软链接/硬链接），并自动重建词法索引。
func KBAdd(opts KBAddOptions) error {
	kb, err := knowledge.Open(opts.Dir)
	if err != nil {
		return err
	}
	if err := kb.Add(opts.Path, opts.LinkMode, opts.Tags); err != nil {
		return err
	}
	// 加入后重建一次（无 embedder，仅词法索引；需要向量请单独执行 KBIndex）
	_, err = kb.BuildIndex(context.Background(), nil)
	return err
}

// KBList 列出知识库中的文档
func KBList(dir string) ([]KnowledgeDoc, error) {
	kb, err := knowledge.Open(dir)
	if err != nil {
		return nil, err
	}
	docs, err := kb.Docs()
	if err != nil {
		return nil, err
	}
	out := make([]KnowledgeDoc, 0, len(docs))
	for _, d := range docs {
		out = append(out, KnowledgeDoc{Name: d.Name, Tags: d.Tags})
	}
	return out, nil
}

// KBIndex 重建知识库索引；配置了 EmbedModel 且厂商支持时附带向量。
func KBIndex(ctx context.Context, opts KBIndexOptions) error {
	kb, err := knowledge.Open(opts.Dir)
	if err != nil {
		return err
	}
	embedder := makeEmbedder(opts.Provider, opts.APIKey, opts.BaseURL, opts.EmbedModel)
	_, err = kb.BuildIndex(ctx, embedder)
	return err
}

// KBSearch 在知识库中检索（向量优先，回退 BM25）。
func KBSearch(ctx context.Context, opts KBSearchOptions) ([]KnowledgeHit, error) {
	kb, err := knowledge.Open(opts.Dir)
	if err != nil {
		return nil, err
	}
	embedder := makeEmbedder(opts.Provider, opts.APIKey, opts.BaseURL, opts.EmbedModel)
	hits, err := kb.Retrieve(ctx, opts.Query, opts.TopK, opts.Selected, embedder)
	if err != nil {
		return nil, err
	}
	out := make([]KnowledgeHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, KnowledgeHit{Doc: h.Doc, Tags: h.Tags, Text: h.Text, Score: h.Score})
	}
	return out, nil
}

// makeEmbedder 据参数构造 Embedder；不满足条件时返回 nil（触发 BM25 回退）。
func makeEmbedder(provider, apiKey, baseURL, embedModel string) llm.Embedder {
	if provider == "" || embedModel == "" {
		return nil
	}
	emb, ok := llm.CreateEmbedder(provider, config.ProviderConfig{APIKey: apiKey, BaseURL: baseURL}, embedModel)
	if !ok {
		return nil
	}
	return emb
}
