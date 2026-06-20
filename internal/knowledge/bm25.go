package knowledge

import (
	"math"
	"sort"
)

// ─────────────────────────────────────────────
// BM25 词法检索（纯 Go，无外部依赖）
//
// 当未配置 embedding 模型时作为 RAG 的检索后端。
// ─────────────────────────────────────────────

type bm25 struct {
	k1    float64
	b     float64
	avgdl float64
	n     int                // 文档(chunk)数
	idf   map[string]float64 // 词 -> 逆文档频率
	tf    []map[string]int   // 每文档词频
	dl    []int              // 每文档长度
}

func newBM25(docTokens [][]string) *bm25 {
	m := &bm25{k1: 1.5, b: 0.75, n: len(docTokens)}
	m.tf = make([]map[string]int, len(docTokens))
	m.dl = make([]int, len(docTokens))
	df := map[string]int{}

	totalLen := 0
	for i, toks := range docTokens {
		freq := map[string]int{}
		for _, t := range toks {
			freq[t]++
		}
		m.tf[i] = freq
		m.dl[i] = len(toks)
		totalLen += len(toks)
		for t := range freq {
			df[t]++
		}
	}
	if m.n > 0 {
		m.avgdl = float64(totalLen) / float64(m.n)
	}

	m.idf = make(map[string]float64, len(df))
	for t, d := range df {
		// 带平滑的 BM25 IDF，保证非负
		m.idf[t] = math.Log(1 + (float64(m.n)-float64(d)+0.5)/(float64(d)+0.5))
	}
	return m
}

// score 计算查询对第 i 个文档的 BM25 得分
func (m *bm25) score(queryTokens []string, i int) float64 {
	if i < 0 || i >= m.n || m.avgdl == 0 {
		return 0
	}
	freq := m.tf[i]
	dl := float64(m.dl[i])
	var s float64
	for _, t := range queryTokens {
		f := float64(freq[t])
		if f == 0 {
			continue
		}
		idf := m.idf[t]
		denom := f + m.k1*(1-m.b+m.b*dl/m.avgdl)
		s += idf * (f * (m.k1 + 1)) / denom
	}
	return s
}

// topK 返回得分最高的 k 个文档下标及分数（降序）
func (m *bm25) topK(queryTokens []string, k int) []scored {
	results := make([]scored, 0, m.n)
	for i := 0; i < m.n; i++ {
		if sc := m.score(queryTokens, i); sc > 0 {
			results = append(results, scored{idx: i, score: sc})
		}
	}
	sort.Slice(results, func(a, b int) bool { return results[a].score > results[b].score })
	if k > 0 && len(results) > k {
		results = results[:k]
	}
	return results
}

type scored struct {
	idx   int
	score float64
}
