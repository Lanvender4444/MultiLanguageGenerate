package config

import (
	"encoding/json"
	"os"
)

type ProviderConfig struct {
	APIKey  string `json:"api_key"`
	Model   string `json:"model"`
	BaseURL string `json:"base_url"`
}

type LLMConfig struct {
	ActiveProvider string                    `json:"active_provider"`
	Providers      map[string]ProviderConfig `json:"providers"`
}

type AppConfig struct {
	Version                 int      `json:"version"`
	LanguageFilePath        string   `json:"language_file_path"`
	OutputDirectory         string   `json:"output_directory"`
	LastSelectedLanguages   []string `json:"last_selected_languages"`
	MaxWorkers              int      `json:"max_workers"`
	RequestTimeoutSeconds   int      `json:"request_timeout_seconds"`
	Theme                   string   `json:"theme"` // "wood" | "silver"
	MergeMode               string   `json:"merge_mode"`   // "none" | "append" | "newfile"
	MergeFormat             string   `json:"merge_format"` // "markdown" | "plain" | "custom"
	MergePrompt             string   `json:"merge_prompt"`
	LLM                     LLMConfig `json:"llm"`
}

func DefaultConfig() *AppConfig {
	return &AppConfig{
		Version:               1,
		LanguageFilePath:      "",
		OutputDirectory:       "",
		LastSelectedLanguages: []string{},
		MaxWorkers:            5,
		RequestTimeoutSeconds: 120,
		Theme:                 "wood",
		MergeMode:             "none",
		MergeFormat:           "markdown",
		MergePrompt:           "",
		LLM: LLMConfig{
			ActiveProvider: "deepseek",
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey:  "",
					Model:   "claude-sonnet-4-6",
					BaseURL: "",
				},
				"openai": {
					APIKey:  "",
					Model:   "gpt-4o",
					BaseURL: "",
				},
				"azure_openai": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"gemini": {
					APIKey:  "",
					Model:   "gemini-2.0-flash",
					BaseURL: "",
				},
				"deepseek": {
					APIKey:  "",
					Model:   "deepseek-chat",
					BaseURL: "",
				},
				"qwen": {
					APIKey:  "",
					Model:   "qwen-max",
					BaseURL: "",
				},
				"zhipu": {
					APIKey:  "",
					Model:   "glm-4-air",
					BaseURL: "",
				},
				"minimax": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"moonshot": {
					APIKey:  "",
					Model:   "moonshot-v1-128k",
					BaseURL: "",
				},
				"mistral": {
					APIKey:  "",
					Model:   "mistral-large-latest",
					BaseURL: "",
				},
				"cohere": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"groq": {
					APIKey:  "",
					Model:   "llama-3.3-70b-versatile",
					BaseURL: "",
				},
				"together": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"perplexity": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"yi": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"baidu": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"hunyuan": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"spark": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"stepfun": {
					APIKey:  "",
					Model:   "",
					BaseURL: "",
				},
				"ollama": {
					APIKey:  "",
					Model:   "llama3",
					BaseURL: "http://localhost:11434",
				},
			},
		},
	}
}

func Load() (*AppConfig, error) {
	path, err := ConfigFilePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := DefaultConfig()
			return cfg, cfg.Save()
		}
		return nil, err
	}

	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *AppConfig) Save() error {
	path, err := ConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
