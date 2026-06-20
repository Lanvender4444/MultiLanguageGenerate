package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
)

// ─────────────────────────────────────────────
// Embeddings 接入
//
// 为 RAG 知识库提供文本向量化能力。仅 OpenAI 兼容厂商实现 /embeddings 接口；
// Anthropic/Gemini 等不支持，调用方应回退到词法(BM25)检索。
// ─────────────────────────────────────────────

// EmbedRequest 一批待向量化的文本
type EmbedRequest struct {
	Input []string
	Model string
}

// Embedder 文本向量化接口
type Embedder interface {
	// Embed 返回与 Input 顺序一致的向量列表
	Embed(ctx context.Context, req EmbedRequest) ([][]float32, error)
	EmbedModel() string
}

// CreateEmbedder 为指定厂商创建 Embedder。
// 仅 OpenAI 兼容厂商支持；其余返回 (nil, false)，调用方据此回退词法检索。
// embedModel 为空时同样返回 false（未配置向量模型）。
func CreateEmbedder(providerID string, pc config.ProviderConfig, embedModel string) (Embedder, bool) {
	if embedModel == "" {
		return nil, false
	}
	info, ok := providerRegistry[providerID]
	if !ok || !info.OpenAICompat {
		return nil, false
	}
	baseURL := pc.BaseURL
	if baseURL == "" {
		baseURL = info.BaseURL
	}
	return &openAIEmbedder{
		baseURL: baseURL,
		apiKey:  pc.APIKey,
		model:   embedModel,
	}, true
}

type openAIEmbedder struct {
	baseURL string
	apiKey  string
	model   string
}

func (e *openAIEmbedder) EmbedModel() string { return e.model }

type embeddingsRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingsResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (e *openAIEmbedder) Embed(ctx context.Context, req EmbedRequest) ([][]float32, error) {
	if len(req.Input) == 0 {
		return nil, nil
	}
	model := req.Model
	if model == "" {
		model = e.model
	}

	body, err := json.Marshal(embeddingsRequest{Model: model, Input: req.Input})
	if err != nil {
		return nil, fmt.Errorf("marshal embeddings request: %w", err)
	}

	url := strings.TrimRight(e.baseURL, "/") + "/embeddings"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+e.apiKey)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed (401): please check your API Key")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embeddings API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var er embeddingsResponse
	if err := json.Unmarshal(respBody, &er); err != nil {
		return nil, fmt.Errorf("parse embeddings response: %w", err)
	}
	if len(er.Data) != len(req.Input) {
		return nil, fmt.Errorf("embeddings count mismatch: got %d, want %d", len(er.Data), len(req.Input))
	}

	// 按 index 字段还原顺序（多数厂商已按序返回，这里做稳妥处理）
	out := make([][]float32, len(req.Input))
	for i, d := range er.Data {
		idx := d.Index
		if idx < 0 || idx >= len(out) {
			idx = i
		}
		out[idx] = d.Embedding
	}
	return out, nil
}
