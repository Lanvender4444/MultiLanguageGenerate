// Package knowledge 实现"专业翻译知识库"（RAG）。
//
// 用户把专业翻译例子、风格指南等文本放进知识库目录（直接拷贝、软链接或
// 硬链接均可），本包负责：读取 → 分块(chunk) → 建立索引（BM25 词法 +
// 可选 embedding 向量）→ 按查询检索最相关片段，供 LLM 参考。
package knowledge

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
)

const (
	indexFileName = "index.json"
	metaFileName  = "meta.json"
	// chunkRunes 单个 chunk 的目标字符数；chunkOverlap 相邻 chunk 的重叠字符数。
	chunkRunes   = 600
	chunkOverlap = 80
)

// 支持读取为文本的扩展名（知识库内容以文本/markdown 为主）
var textExts = map[string]bool{
	".txt": true, ".md": true, ".markdown": true, ".text": true,
	".json": true, ".csv": true, ".tsv": true, ".rst": true, ".log": true,
	".po": true, ".srt": true, ".vtt": true, ".html": true, ".xml": true, ".yaml": true, ".yml": true,
}

// KnowledgeBase 指向一个知识库目录
type KnowledgeBase struct {
	Dir string
}

// DocMeta 知识库中一个文档的元信息
type DocMeta struct {
	Name string   `json:"name"` // 知识库目录内的文件名
	Tags []string `json:"tags,omitempty"`
}

// IndexChunk 索引中的一个文本片段
type IndexChunk struct {
	Doc       string    `json:"doc"`
	Tags      []string  `json:"tags,omitempty"`
	Text      string    `json:"text"`
	Embedding []float32 `json:"embedding,omitempty"`
}

// Index 持久化的知识库索引
type Index struct {
	EmbedModel string       `json:"embed_model,omitempty"`
	Chunks     []IndexChunk `json:"chunks"`
}

// Hit 一条检索结果
type Hit struct {
	Doc   string
	Tags  []string
	Text  string
	Score float64
}

// DefaultDir 返回默认知识库目录：<配置目录>/knowledge
func DefaultDir() (string, error) {
	base, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "knowledge"), nil
}

// Open 打开（必要时创建）知识库目录
func Open(dir string) (*KnowledgeBase, error) {
	if dir == "" {
		d, err := DefaultDir()
		if err != nil {
			return nil, err
		}
		dir = d
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create knowledge dir: %w", err)
	}
	return &KnowledgeBase{Dir: dir}, nil
}

// ─────────────────────────────────────────────
// 添加内容（拷贝 / 软链接 / 硬链接）
// ─────────────────────────────────────────────

// Add 把 srcPath 指向的文件加入知识库。
// linkMode: "copy"(默认) | "soft"(符号链接) | "hard"(硬链接)。
// tags 为可选分类标签（如 "style"、"legal"）。
func (kb *KnowledgeBase) Add(srcPath, linkMode string, tags []string) error {
	abs, err := filepath.Abs(srcPath)
	if err != nil {
		return err
	}
	if _, err := os.Stat(abs); err != nil {
		return fmt.Errorf("source not accessible: %w", err)
	}
	dst := filepath.Join(kb.Dir, filepath.Base(abs))

	// 目标已存在则先移除（允许覆盖更新）
	_ = os.Remove(dst)

	switch linkMode {
	case "soft", "symlink":
		if err := os.Symlink(abs, dst); err != nil {
			return fmt.Errorf("create symlink: %w", err)
		}
	case "hard", "link":
		if err := os.Link(abs, dst); err != nil {
			return fmt.Errorf("create hardlink: %w", err)
		}
	default: // copy
		if err := copyFile(abs, dst); err != nil {
			return fmt.Errorf("copy file: %w", err)
		}
	}

	// 记录标签
	meta, _ := kb.loadMeta()
	meta[filepath.Base(abs)] = tags
	return kb.saveMeta(meta)
}

// Remove 从知识库移除一个文档（按文件名）
func (kb *KnowledgeBase) Remove(name string) error {
	_ = os.Remove(filepath.Join(kb.Dir, name))
	meta, _ := kb.loadMeta()
	delete(meta, name)
	return kb.saveMeta(meta)
}

// Docs 列出知识库中的全部文档（含标签）
func (kb *KnowledgeBase) Docs() ([]DocMeta, error) {
	entries, err := os.ReadDir(kb.Dir)
	if err != nil {
		return nil, err
	}
	meta, _ := kb.loadMeta()
	var out []DocMeta
	for _, e := range entries {
		name := e.Name()
		if name == indexFileName || name == metaFileName {
			continue
		}
		// e.IsDir() 对符号链接指向的目录可能为 false/true；只收文本文件
		if !textExts[strings.ToLower(filepath.Ext(name))] {
			continue
		}
		out = append(out, DocMeta{Name: name, Tags: meta[name]})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// ─────────────────────────────────────────────
// 建立 / 读取索引
// ─────────────────────────────────────────────

// BuildIndex 重新扫描全部文档，分块并（在 embedder 可用时）计算向量，写入 index.json。
func (kb *KnowledgeBase) BuildIndex(ctx context.Context, embedder llm.Embedder) (*Index, error) {
	docs, err := kb.Docs()
	if err != nil {
		return nil, err
	}

	idx := &Index{}
	var texts []string // 待向量化的 chunk 文本（与 idx.Chunks 顺序一致）

	for _, d := range docs {
		content, rerr := readTextFile(filepath.Join(kb.Dir, d.Name))
		if rerr != nil {
			continue // 跳过无法读取的文件
		}
		for _, c := range chunkText(content) {
			idx.Chunks = append(idx.Chunks, IndexChunk{Doc: d.Name, Tags: d.Tags, Text: c})
			texts = append(texts, c)
		}
	}

	// 可选：计算向量
	if embedder != nil && len(texts) > 0 {
		idx.EmbedModel = embedder.EmbedModel()
		vecs, eerr := embedAll(ctx, embedder, texts)
		if eerr != nil {
			// 向量化失败不致命：保留纯文本索引，检索时自动回退 BM25
			idx.EmbedModel = ""
		} else {
			for i := range idx.Chunks {
				if i < len(vecs) {
					idx.Chunks[i].Embedding = vecs[i]
				}
			}
		}
	}

	if err := kb.saveIndex(idx); err != nil {
		return nil, err
	}
	return idx, nil
}

// LoadIndex 读取已保存的索引；不存在则返回空索引。
func (kb *KnowledgeBase) LoadIndex() (*Index, error) {
	data, err := os.ReadFile(filepath.Join(kb.Dir, indexFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return &Index{}, nil
		}
		return nil, err
	}
	var idx Index
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// ─────────────────────────────────────────────
// 检索（混合：有向量用向量，否则 BM25）
// ─────────────────────────────────────────────

// Retrieve 返回与 query 最相关的 topK 个片段。
// selected 非空时只在指定的文档名或标签范围内检索。
// embedder 非 nil 且索引含向量时走余弦相似度，否则回退 BM25。
func (kb *KnowledgeBase) Retrieve(ctx context.Context, query string, topK int, selected []string, embedder llm.Embedder) ([]Hit, error) {
	idx, err := kb.LoadIndex()
	if err != nil {
		return nil, err
	}
	chunks := filterChunks(idx.Chunks, selected)
	if len(chunks) == 0 {
		return nil, nil
	}
	if topK <= 0 {
		topK = 5
	}

	// 路径一：向量检索
	if embedder != nil && idx.EmbedModel != "" && allHaveEmbedding(chunks) {
		qv, eerr := embedder.Embed(ctx, llm.EmbedRequest{Input: []string{query}})
		if eerr == nil && len(qv) == 1 && len(qv[0]) > 0 {
			scoredList := make([]scored, 0, len(chunks))
			for i, c := range chunks {
				scoredList = append(scoredList, scored{idx: i, score: float64(cosine(qv[0], c.Embedding))})
			}
			sort.Slice(scoredList, func(a, b int) bool { return scoredList[a].score > scoredList[b].score })
			return toHits(chunks, scoredList, topK), nil
		}
		// 向量化失败则继续走 BM25
	}

	// 路径二：BM25 词法检索
	docTokens := make([][]string, len(chunks))
	for i, c := range chunks {
		docTokens[i] = tokenize(c.Text)
	}
	m := newBM25(docTokens)
	res := m.topK(tokenize(query), topK)
	return toHits(chunks, res, topK), nil
}

// ─────────────────────────────────────────────
// 内部辅助
// ─────────────────────────────────────────────

func toHits(chunks []IndexChunk, scoredList []scored, topK int) []Hit {
	if topK > 0 && len(scoredList) > topK {
		scoredList = scoredList[:topK]
	}
	hits := make([]Hit, 0, len(scoredList))
	for _, s := range scoredList {
		c := chunks[s.idx]
		hits = append(hits, Hit{Doc: c.Doc, Tags: c.Tags, Text: c.Text, Score: s.score})
	}
	return hits
}

func filterChunks(chunks []IndexChunk, selected []string) []IndexChunk {
	if len(selected) == 0 {
		return chunks
	}
	sel := map[string]bool{}
	for _, s := range selected {
		sel[s] = true
	}
	var out []IndexChunk
	for _, c := range chunks {
		if sel[c.Doc] {
			out = append(out, c)
			continue
		}
		for _, t := range c.Tags {
			if sel[t] {
				out = append(out, c)
				break
			}
		}
	}
	return out
}

func allHaveEmbedding(chunks []IndexChunk) bool {
	for _, c := range chunks {
		if len(c.Embedding) == 0 {
			return false
		}
	}
	return true
}

// chunkText 按段落聚合切块，带重叠，避免切断语义。
func chunkText(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	paras := strings.Split(s, "\n")
	var chunks []string
	var cur []rune
	flush := func() {
		if len(strings.TrimSpace(string(cur))) > 0 {
			chunks = append(chunks, strings.TrimSpace(string(cur)))
		}
		cur = nil
	}
	for _, p := range paras {
		pr := []rune(p + "\n")
		if len(cur)+len(pr) > chunkRunes && len(cur) > 0 {
			flush()
			// 重叠：保留上一块尾部
			if len(chunks) > 0 {
				prev := []rune(chunks[len(chunks)-1])
				if len(prev) > chunkOverlap {
					cur = append(cur, prev[len(prev)-chunkOverlap:]...)
				}
			}
		}
		cur = append(cur, pr...)
	}
	flush()
	return chunks
}

// embedAll 分批向量化（避免单请求过大）
func embedAll(ctx context.Context, embedder llm.Embedder, texts []string) ([][]float32, error) {
	const batch = 64
	out := make([][]float32, 0, len(texts))
	for i := 0; i < len(texts); i += batch {
		end := i + batch
		if end > len(texts) {
			end = len(texts)
		}
		vecs, err := embedder.Embed(ctx, llm.EmbedRequest{Input: texts[i:end]})
		if err != nil {
			return nil, err
		}
		out = append(out, vecs...)
	}
	return out, nil
}

func cosine(a, b []float32) float32 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return float32(dot / (math.Sqrt(na) * math.Sqrt(nb)))
}

// ─────────────────────────────────────────────
// 文件 / 元数据 IO
// ─────────────────────────────────────────────

func (kb *KnowledgeBase) saveIndex(idx *Index) error {
	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(kb.Dir, indexFileName), data, 0644)
}

func (kb *KnowledgeBase) loadMeta() (map[string][]string, error) {
	data, err := os.ReadFile(filepath.Join(kb.Dir, metaFileName))
	if err != nil {
		return map[string][]string{}, nil
	}
	m := map[string][]string{}
	_ = json.Unmarshal(data, &m)
	return m, nil
}

func (kb *KnowledgeBase) saveMeta(m map[string][]string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(kb.Dir, metaFileName), data, 0644)
}

func readTextFile(path string) (string, error) {
	data, err := os.ReadFile(path) // 自动跟随软链接/硬链接
	if err != nil {
		return "", err
	}
	// 去 UTF-8 BOM
	if len(data) >= 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	return string(data), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
