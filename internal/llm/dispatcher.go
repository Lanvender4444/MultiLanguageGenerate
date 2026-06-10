package llm

import (
	"fmt"

	"github.com/yourname/MultiLanguageGenerate/internal/config"
)

type ProviderInfo struct {
	ID          string
	DisplayName string
	BaseURL     string
	OpenAICompat bool
}

var providerRegistry = map[string]ProviderInfo{
	"anthropic":    {ID: "anthropic", DisplayName: "Claude (Anthropic)", BaseURL: "https://api.anthropic.com/v1", OpenAICompat: false},
	"openai":       {ID: "openai", DisplayName: "OpenAI", BaseURL: "https://api.openai.com/v1", OpenAICompat: true},
	"azure_openai": {ID: "azure_openai", DisplayName: "Azure OpenAI", BaseURL: "", OpenAICompat: true},
	"gemini":       {ID: "gemini", DisplayName: "Google Gemini", BaseURL: "https://generativelanguage.googleapis.com", OpenAICompat: false},
	"deepseek":     {ID: "deepseek", DisplayName: "DeepSeek", BaseURL: "https://api.deepseek.com/v1", OpenAICompat: true},
	"qwen":         {ID: "qwen", DisplayName: "通义千问 (Qwen)", BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", OpenAICompat: true},
	"zhipu":        {ID: "zhipu", DisplayName: "智谱 GLM (Zhipu)", BaseURL: "https://open.bigmodel.cn/api/paas/v4", OpenAICompat: true},
	"minimax":      {ID: "minimax", DisplayName: "MiniMax", BaseURL: "https://api.minimax.chat/v1", OpenAICompat: true},
	"moonshot":     {ID: "moonshot", DisplayName: "Moonshot / Kimi", BaseURL: "https://api.moonshot.cn/v1", OpenAICompat: true},
	"mistral":      {ID: "mistral", DisplayName: "Mistral AI", BaseURL: "https://api.mistral.ai/v1", OpenAICompat: true},
	"cohere":       {ID: "cohere", DisplayName: "Cohere", BaseURL: "https://api.cohere.com/v1", OpenAICompat: true},
	"groq":         {ID: "groq", DisplayName: "Groq", BaseURL: "https://api.groq.com/openai/v1", OpenAICompat: true},
	"together":     {ID: "together", DisplayName: "Together AI", BaseURL: "https://api.together.xyz/v1", OpenAICompat: true},
	"perplexity":   {ID: "perplexity", DisplayName: "Perplexity AI", BaseURL: "https://api.perplexity.ai", OpenAICompat: true},
	"yi":           {ID: "yi", DisplayName: "零一万物 (Yi)", BaseURL: "https://api.lingyiwanwu.com/v1", OpenAICompat: true},
	"baidu":        {ID: "baidu", DisplayName: "文心一言 (ERNIE)", BaseURL: "https://aip.baidubce.com", OpenAICompat: false},
	"hunyuan":      {ID: "hunyuan", DisplayName: "混元 (Tencent)", BaseURL: "https://hunyuan.tencentcloudapi.com", OpenAICompat: false},
	"spark":        {ID: "spark", DisplayName: "讯飞星火 (Spark)", BaseURL: "", OpenAICompat: false},
	"stepfun":      {ID: "stepfun", DisplayName: "阶跃星辰 (StepFun)", BaseURL: "https://api.stepfun.com/v1", OpenAICompat: true},
	"ollama":       {ID: "ollama", DisplayName: "Ollama (本地)", BaseURL: "http://localhost:11434/v1", OpenAICompat: true},
}

func GetProviderInfo(id string) (ProviderInfo, bool) {
	info, ok := providerRegistry[id]
	return info, ok
}

func AllProviders() []ProviderInfo {
	ids := []string{
		"anthropic", "openai", "azure_openai", "gemini", "deepseek",
		"qwen", "zhipu", "minimax", "moonshot", "mistral",
		"cohere", "groq", "together", "perplexity", "yi",
		"baidu", "hunyuan", "spark", "stepfun", "ollama",
	}
	result := make([]ProviderInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := providerRegistry[id]; ok {
			result = append(result, info)
		}
	}
	return result
}

func CreateProvider(providerID string, pc config.ProviderConfig) (Provider, error) {
	info, ok := providerRegistry[providerID]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", providerID)
	}

	baseURL := pc.BaseURL
	if baseURL == "" {
		baseURL = info.BaseURL
	}

	switch providerID {
	case "anthropic":
		return &AnthropicProvider{
			APIKey:  pc.APIKey,
			Model:   pc.Model,
			BaseURL: pc.BaseURL,
		}, nil
	case "gemini":
		return &GeminiProvider{
			APIKey:  pc.APIKey,
			Model:   pc.Model,
			BaseURL: pc.BaseURL,
		}, nil
	default:
		if info.OpenAICompat {
			return &OpenAICompatProvider{
				BaseURL:      baseURL,
				APIKey:       pc.APIKey,
				ProviderName: info.ID,
				Model:        pc.Model,
			}, nil
		}
		return nil, fmt.Errorf("provider %s is not yet implemented", providerID)
	}
}
