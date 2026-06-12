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

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/processor"
)

type GeminiProvider struct {
	APIKey  string
	Model   string
	BaseURL string
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiSystemInstruction struct {
	Parts []geminiPart `json:"parts"`
}

type geminiGenerateRequest struct {
	Contents         []geminiContent          `json:"contents"`
	SystemInstruction *geminiSystemInstruction `json:"systemInstruction,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
}

func (p *GeminiProvider) Translate(ctx context.Context, req TranslateRequest) (string, error) {
	sysPrompt := req.SystemPrompt
	if sysPrompt == "" {
		sysPrompt = processor.BuildSystemPrompt(req.SourceLanguage, req.TargetLanguage, req.TargetCode, req.SourceType)
	}

	genReq := geminiGenerateRequest{
		SystemInstruction: &geminiSystemInstruction{
			Parts: []geminiPart{{Text: sysPrompt}},
		},
		Contents: []geminiContent{
			{
				Role:  "user",
				Parts: []geminiPart{{Text: req.SourceText}},
			},
		},
	}

	body, err := json.Marshal(genReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	model := req.Model
	if model == "" {
		model = "gemini-2.0-flash"
	}

	baseURL := "https://generativelanguage.googleapis.com"
	if p.BaseURL != "" {
		baseURL = strings.TrimRight(p.BaseURL, "/")
	}
	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", baseURL, model, p.APIKey)

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 180 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode == 401 || resp.StatusCode == 403 {
		return "", fmt.Errorf("authentication failed (%d): please check your API Key", resp.StatusCode)
	}
	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429): too many requests")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBody, &gemResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return gemResp.Candidates[0].Content.Parts[0].Text, nil
}

func (p *GeminiProvider) ListModels(_ context.Context) ([]string, error) {
	return []string{
		"gemini-2.0-flash",
		"gemini-2.5-pro",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}, nil
}

func (p *GeminiProvider) Name() string {
	return "gemini"
}
