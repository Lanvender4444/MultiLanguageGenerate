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

type OpenAICompatProvider struct {
	BaseURL      string
	APIKey       string
	ProviderName string
	Model        string
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

type modelsResponse struct {
	Data []modelEntry `json:"data"`
}

type modelEntry struct {
	ID string `json:"id"`
}

func buildSystemPrompt(srcLang, dstLang, dstCode string) string {
	return fmt.Sprintf(`You are a professional technical document translator.
Translate the following document from %s to %s (%s).

Rules:
1. Preserve ALL original formatting (Markdown syntax, HTML tags, code blocks, etc.)
2. Do NOT translate content inside code blocks (`+"```"+` `+"```"+` or indented code)
3. Do NOT translate URLs, file paths, variable names, or command-line arguments
4. Translate comments inside code blocks if they are in the source language
5. Keep the same line structure and whitespace as the original
6. Output ONLY the translated content, no explanations or preamble`, srcLang, dstLang, dstCode)
}

func (p *OpenAICompatProvider) Translate(ctx context.Context, req TranslateRequest) (string, error) {
	sysPrompt := buildSystemPrompt(req.SourceLanguage, req.TargetLanguage, req.TargetCode)

	chatReq := chatRequest{
		Model: req.Model,
		Messages: []chatMessage{
			{Role: "system", Content: sysPrompt},
			{Role: "user", Content: req.SourceText},
		},
	}

	body, err := json.Marshal(chatReq)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	baseURL := strings.TrimRight(p.BaseURL, "/")
	url := baseURL + "/chat/completions"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

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

	var chatResp chatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices in response")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (p *OpenAICompatProvider) ListModels(ctx context.Context) ([]string, error) {
	baseURL := strings.TrimRight(p.BaseURL, "/")
	url := baseURL + "/models"

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error (%d)", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var modelsResp modelsResponse
	if err := json.Unmarshal(body, &modelsResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	models := make([]string, 0, len(modelsResp.Data))
	for _, m := range modelsResp.Data {
		models = append(models, m.ID)
	}

	return models, nil
}

func (p *OpenAICompatProvider) Name() string {
	return p.ProviderName
}
