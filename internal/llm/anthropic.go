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
)

type AnthropicProvider struct {
	APIKey  string
	Model   string
	BaseURL string
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicResponse struct {
	Content []anthropicContentBlock `json:"content"`
}

func (p *AnthropicProvider) Translate(ctx context.Context, req TranslateRequest) (string, error) {
	sysPrompt := buildSystemPrompt(req.SourceLanguage, req.TargetLanguage, req.TargetCode)

	antReq := anthropicRequest{
		Model:     req.Model,
		MaxTokens: 8192,
		System:    sysPrompt,
		Messages: []anthropicMessage{
			{Role: "user", Content: req.SourceText},
		},
	}

	body, err := json.Marshal(antReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	baseURL := "https://api.anthropic.com/v1"
	if p.BaseURL != "" {
		baseURL = strings.TrimRight(p.BaseURL, "/")
	}
	url := baseURL + "/messages"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

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

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("authentication failed (401): please check your API Key")
	}
	if resp.StatusCode == 429 {
		return "", fmt.Errorf("rate limited (429): too many requests")
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	var antResp anthropicResponse
	if err := json.Unmarshal(respBody, &antResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(antResp.Content) == 0 {
		return "", fmt.Errorf("no content in response")
	}

	return antResp.Content[0].Text, nil
}

func (p *AnthropicProvider) ListModels(_ context.Context) ([]string, error) {
	return []string{
		"claude-sonnet-4-6",
		"claude-haiku-4-5",
		"claude-opus-4-6",
		"claude-3-5-sonnet-20241022",
		"claude-3-5-haiku-20241022",
		"claude-3-opus-20240229",
	}, nil
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}
