// Package translate 是 MultiLanguageGenerate 的公共高层 API。
//
// 它把内部的文件解析（processor）、LLM 接入（llm）与并发引擎（translator）
// 封装成一个对外可 import 的稳定接口，供其它 Go 程序或命令行工具复用：
//
//	import "github.com/Lanvender4444/MultiLanguageGenerate/translate"
//
//	results, err := translate.Run(context.Background(), translate.Options{
//	    SourceFile:  "report.docx",
//	    TargetCodes: []string{"en", "ja"},
//	    Provider:    "deepseek",
//	    Model:       "deepseek-chat",
//	    APIKey:      "sk-...",
//	})
//
// 该包只依赖纯 Go 的内部包，不引入 GUI(Fyne)/CGO 依赖，可在服务器、CI 等
// 无图形环境中使用。
package translate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/detector"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/language"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/translator"
)

// 默认值
const (
	DefaultMaxWorkers = 5
	DefaultTimeout    = 120 * time.Second
)

// Options 描述一次翻译任务的全部输入。
type Options struct {
	// SourceFile 待翻译的源文件路径（必填）。
	SourceFile string
	// TargetCodes 目标语言代码列表，如 []string{"en","ja","fr"}（必填，至少一个）。
	TargetCodes []string
	// SourceLanguage 源语言显示名；留空或 "auto" 时按内容本地自动识别。
	SourceLanguage string
	// OutputDir 输出目录；留空则与源文件同目录。
	OutputDir string

	// Provider LLM 厂商 ID，如 "anthropic"、"deepseek"、"openai"（必填）。
	Provider string
	// Model 模型名；留空时由厂商决定默认值。
	Model string
	// APIKey 厂商 API Key；provider 为 "ollama" 时可留空。
	APIKey string
	// BaseURL 自定义服务地址；留空使用厂商默认。
	BaseURL string

	// MaxWorkers 并发翻译的最大协程数；<=0 时取 DefaultMaxWorkers。
	MaxWorkers int
	// Timeout 单次 LLM 请求超时；<=0 时取 DefaultTimeout。
	Timeout time.Duration

	// OnResult 可选回调：每完成一种目标语言即被调用一次（用于流式进度展示）。
	OnResult func(FileResult)
}

// FileResult 是单种目标语言的翻译结果。
type FileResult struct {
	TargetCode string // 目标语言代码
	TargetName string // 目标语言显示名
	OutputPath string // 成功时的输出文件路径
	Err        error  // 失败时的错误，成功为 nil
}

// ProviderInfo 描述一个可用的 LLM 厂商。
type ProviderInfo struct {
	ID          string
	DisplayName string
	BaseURL     string
	NeedsKey    bool // 是否需要 API Key（ollama 为 false）
}

// Language 描述一种语言。
type Language struct {
	Code string
	Name string
}

// Run 执行翻译。它会为每个目标语言并发生成一份保留原格式的译文文件，
// 返回与 TargetCodes 对应的结果列表。
//
// 返回的 error 仅表示"任务无法启动"的前置错误（如参数缺失、厂商未知）；
// 单个语言翻译失败不会让整体 error 非空，而是体现在对应 FileResult.Err 上。
func Run(ctx context.Context, opts Options) ([]FileResult, error) {
	// ── 参数校验 ──
	if opts.SourceFile == "" {
		return nil, fmt.Errorf("translate: SourceFile is required")
	}
	if _, err := os.Stat(opts.SourceFile); err != nil {
		return nil, fmt.Errorf("translate: cannot access source file: %w", err)
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

	// ── 构造 Provider ──
	provider, err := llm.CreateProvider(opts.Provider, config.ProviderConfig{
		APIKey:  opts.APIKey,
		Model:   opts.Model,
		BaseURL: opts.BaseURL,
	})
	if err != nil {
		return nil, fmt.Errorf("translate: create provider: %w", err)
	}

	// ── 源语言 ──
	srcLang := opts.SourceLanguage
	if srcLang == "" || srcLang == "auto" {
		if head, rerr := readHead(opts.SourceFile, 8192); rerr == nil {
			_, srcLang = detector.DetectLocal(string(head))
		} else {
			srcLang = "English"
		}
	}

	// ── 文件类型与输出目录 ──
	srcType := filetype.DetectFile(opts.SourceFile)
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = filepath.Dir(opts.SourceFile)
	}

	// ── 引擎参数 ──
	workers := opts.MaxWorkers
	if workers <= 0 {
		workers = DefaultMaxWorkers
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}

	engine := translator.NewEngine(provider, opts.Model, workers, timeout)

	// ── 构造 Job 列表（记住每个 code 的显示名供结果回填）──
	names := make(map[string]string, len(opts.TargetCodes))
	jobs := make([]translator.Job, 0, len(opts.TargetCodes))
	for _, code := range opts.TargetCodes {
		name := language.NameByCode(code)
		names[code] = name
		jobs = append(jobs, translator.Job{
			SourceFile:     opts.SourceFile,
			SourceLanguage: srcLang,
			TargetCode:     code,
			TargetName:     name,
			OutputDir:      outputDir,
			SourceFileType: srcType,
		})
	}

	// ── 并发执行并收集结果 ──
	progress := make(chan translator.Result, len(jobs))
	go engine.Run(ctx, jobs, progress)

	results := make([]FileResult, 0, len(jobs))
	for r := range progress {
		fr := FileResult{
			TargetCode: r.TargetCode,
			TargetName: names[r.TargetCode],
			OutputPath: r.OutputPath,
			Err:        r.Error,
		}
		if opts.OnResult != nil {
			opts.OnResult(fr)
		}
		results = append(results, fr)
	}

	return results, nil
}

// Providers 返回全部可用的 LLM 厂商（顺序稳定，供列表展示）。
func Providers() []ProviderInfo {
	src := llm.AllProviders()
	out := make([]ProviderInfo, 0, len(src))
	for _, p := range src {
		out = append(out, ProviderInfo{
			ID:          p.ID,
			DisplayName: p.DisplayName,
			BaseURL:     p.BaseURL,
			NeedsKey:    p.ID != "ollama",
		})
	}
	return out
}

// Languages 返回内嵌的全部可用语言（按代码排序）。
func Languages() ([]Language, error) {
	langs, err := language.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	out := make([]Language, 0, len(langs))
	for _, l := range langs {
		out = append(out, Language{Code: l.Code, Name: l.Name})
	}
	return out, nil
}

// FileTypeName 返回源文件被识别出的类型描述（如 "Word Document"）。
func FileTypeName(path string) string {
	return filetype.TypeInfoOf(filetype.DetectFile(path)).Description
}

// readHead 读取文件前 n 字节，用于本地源语言识别。
func readHead(path string, n int) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, n)
	read, err := f.Read(buf)
	if err != nil && read == 0 {
		return nil, err
	}
	return buf[:read], nil
}
