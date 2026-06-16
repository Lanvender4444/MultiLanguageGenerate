// Command mlg-cli 是 MultiLanguageGenerate 的命令行版本。
//
// 它复用与 GUI 相同的配置文件（config.json），并允许用命令行参数临时覆盖
// provider / model / key 等设置，适合脚本、批处理与 CI 场景。
//
// 示例：
//
//	# 列出可用语言 / 厂商
//	mlg-cli -list-langs
//	mlg-cli -list-providers
//
//	# 把 report.docx 翻译成英语和日语（复用 config.json 里已保存的厂商与 Key）
//	mlg-cli -file report.docx -to en,ja
//
//	# 临时指定厂商与 Key，覆盖配置
//	mlg-cli -file notes.md -to fr,de -provider deepseek -model deepseek-chat -key sk-xxx
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/translate"
)

func main() {
	var (
		file         = flag.String("file", "", "源文件路径（必填）")
		to           = flag.String("to", "", "目标语言代码，逗号分隔，如 en,ja,fr（必填）")
		from         = flag.String("from", "auto", "源语言显示名；auto=按内容自动识别")
		out          = flag.String("out", "", "输出目录；留空则与源文件同目录")
		provider     = flag.String("provider", "", "LLM 厂商 ID；留空则用 config.json 中的 active_provider")
		model        = flag.String("model", "", "模型名；留空则用配置中的默认值")
		key          = flag.String("key", "", "API Key；留空则用配置/环境变量 MLG_API_KEY")
		baseURL      = flag.String("base-url", "", "自定义服务地址；留空使用厂商默认")
		workers      = flag.Int("workers", 0, "并发数；<=0 则用配置值或默认 5")
		timeoutSec   = flag.Int("timeout", 0, "单请求超时(秒)；<=0 则用配置值或默认 120")
		listLangs    = flag.Bool("list-langs", false, "打印全部可用语言代码后退出")
		listProvider = flag.Bool("list-providers", false, "打印全部可用厂商后退出")
	)
	flag.Parse()

	if *listLangs {
		printLanguages()
		return
	}
	if *listProvider {
		printProviders()
		return
	}

	// ── 基本参数校验 ──
	if *file == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "错误：-file 和 -to 为必填参数。")
		fmt.Fprintln(os.Stderr, "用法示例：mlg-cli -file report.docx -to en,ja")
		flag.Usage()
		os.Exit(2)
	}

	targetCodes := splitCSV(*to)
	if len(targetCodes) == 0 {
		fmt.Fprintln(os.Stderr, "错误：-to 未解析出有效语言代码。")
		os.Exit(2)
	}

	// ── 读取配置（失败则用默认值，不阻断 CLI）──
	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// ── 解析厂商配置：config + 命令行覆盖 ──
	providerID := *provider
	if providerID == "" {
		providerID = cfg.LLM.ActiveProvider
	}
	pc := cfg.LLM.Providers[providerID] // 不存在则为零值

	apiKey := firstNonEmpty(*key, os.Getenv("MLG_API_KEY"), pc.APIKey)
	modelName := firstNonEmpty(*model, pc.Model)
	base := firstNonEmpty(*baseURL, pc.BaseURL)

	maxWorkers := *workers
	if maxWorkers <= 0 {
		maxWorkers = cfg.MaxWorkers
	}
	timeout := time.Duration(*timeoutSec) * time.Second
	if *timeoutSec <= 0 && cfg.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	}

	outputDir := firstNonEmpty(*out, cfg.OutputDirectory)

	// ── 任务前信息 ──
	fmt.Printf("源文件   : %s (%s)\n", *file, translate.FileTypeName(*file))
	fmt.Printf("源语言   : %s\n", *from)
	fmt.Printf("目标语言 : %s\n", strings.Join(targetCodes, ", "))
	fmt.Printf("厂商/模型: %s / %s\n", providerID, fallback(modelName, "(默认)"))
	fmt.Println("开始翻译...")

	// ── 执行（流式打印每种语言的结果）──
	total := len(targetCodes)
	done := 0
	failed := 0
	results, runErr := translate.Run(context.Background(), translate.Options{
		SourceFile:     *file,
		TargetCodes:    targetCodes,
		SourceLanguage: *from,
		OutputDir:      outputDir,
		Provider:       providerID,
		Model:          modelName,
		APIKey:         apiKey,
		BaseURL:        base,
		MaxWorkers:     maxWorkers,
		Timeout:        timeout,
		OnResult: func(r translate.FileResult) {
			done++
			label := fmt.Sprintf("%s (%s)", r.TargetCode, r.TargetName)
			if r.Err != nil {
				failed++
				fmt.Printf("  [%d/%d] ✗ %-22s 失败: %v\n", done, total, label, r.Err)
			} else {
				fmt.Printf("  [%d/%d] ✓ %-22s → %s\n", done, total, label, r.OutputPath)
			}
		},
	})

	if runErr != nil {
		fmt.Fprintf(os.Stderr, "无法启动翻译：%v\n", runErr)
		os.Exit(1)
	}

	ok := len(results) - failed
	fmt.Printf("\n完成：成功 %d，失败 %d，共 %d。\n", ok, failed, len(results))
	if failed > 0 {
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────
// 辅助
// ─────────────────────────────────────────────

func printLanguages() {
	langs, err := translate.Languages()
	if err != nil {
		fmt.Fprintf(os.Stderr, "无法加载语言列表：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("可用语言（共 %d 种）：\n", len(langs))
	for _, l := range langs {
		fmt.Printf("  %-8s %s\n", l.Code, l.Name)
	}
}

func printProviders() {
	provs := translate.Providers()
	fmt.Printf("可用厂商（共 %d 家）：\n", len(provs))
	for _, p := range provs {
		keyHint := ""
		if !p.NeedsKey {
			keyHint = "  (无需 Key)"
		}
		fmt.Printf("  %-14s %s%s\n", p.ID, p.DisplayName, keyHint)
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func fallback(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
