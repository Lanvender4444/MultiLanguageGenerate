// Command mlg-cli 是 MultiLanguageGenerate 的命令行版本。
//
// 子命令：
//
//	mlg-cli translate ...   翻译文件（默认命令，可省略 "translate"）
//	mlg-cli kb ...          管理专业翻译知识库（RAG）
//	mlg-cli glossary ...    生成 / 查看专业名词词表
//
// 配置来源优先级：命令行参数 > 环境变量(MLG_API_KEY / MLG_EMBED_MODEL) > config.json。
// config.json 与 GUI 共用。
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
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(2)
	}

	switch args[0] {
	case "kb":
		runKB(args[1:])
	case "glossary", "gloss":
		runGlossary(args[1:])
	case "translate":
		runTranslate(args[1:])
	case "-h", "--help", "help":
		usage()
	default:
		// 向后兼容：以 "-" 开头的首参视为旧版 translate 用法
		if strings.HasPrefix(args[0], "-") {
			runTranslate(args)
		} else {
			fmt.Fprintf(os.Stderr, "未知命令：%q\n\n", args[0])
			usage()
			os.Exit(2)
		}
	}
}

func usage() {
	fmt.Println(`mlg-cli — 多语言多格式 AI 翻译工具

用法：
  mlg-cli translate -file <路径> -to <代码,...> [选项]   翻译文件
  mlg-cli kb <add|list|index|search> [选项]              管理知识库(RAG)
  mlg-cli glossary <gen|show> [选项]                     专业名词词表

常用示例：
  mlg-cli -file report.docx -to en,ja
  mlg-cli kb add ./style.md -tags style -link soft
  mlg-cli kb index -embed-model text-embedding-3-small
  mlg-cli glossary gen -prompt "提取人名与术语" -to en,ja -o glossary.json
  mlg-cli translate -file novel.md -to en -glossary glossary.json

各子命令的完整参数见： mlg-cli <子命令> -h`)
}

// ─────────────────────────────────────────────
// translate 子命令
// ─────────────────────────────────────────────

func runTranslate(argv []string) {
	fs := flag.NewFlagSet("translate", flag.ExitOnError)
	var (
		file         = fs.String("file", "", "源文件路径（必填）")
		to           = fs.String("to", "", "目标语言代码，逗号分隔，如 en,ja,fr（必填）")
		from         = fs.String("from", "auto", "源语言显示名；auto=自动识别")
		out          = fs.String("out", "", "输出目录；留空则与源文件同目录")
		glossaryPath = fs.String("glossary", "", "专业名词词表 JSON 路径（可选，启用术语约束）")
		provider     = fs.String("provider", "", "厂商 ID；留空用 config.json 的 active_provider")
		model        = fs.String("model", "", "模型名")
		key          = fs.String("key", "", "API Key")
		baseURL      = fs.String("base-url", "", "自定义服务地址")
		workers      = fs.Int("workers", 0, "并发数；<=0 用配置或默认 5")
		timeoutSec   = fs.Int("timeout", 0, "单请求超时(秒)；<=0 用配置或默认 120")
		listLangs    = fs.Bool("list-langs", false, "打印全部语言后退出")
		listProvider = fs.Bool("list-providers", false, "打印全部厂商后退出")
	)
	_ = fs.Parse(argv)

	if *listLangs {
		printLanguages()
		return
	}
	if *listProvider {
		printProviders()
		return
	}

	if *file == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "错误：-file 和 -to 为必填参数。")
		fmt.Fprintln(os.Stderr, "示例：mlg-cli -file report.docx -to en,ja")
		os.Exit(2)
	}

	targetCodes := splitCSV(*to)
	if len(targetCodes) == 0 {
		fmt.Fprintln(os.Stderr, "错误：-to 未解析出有效语言代码。")
		os.Exit(2)
	}

	cfg := loadConfig()
	providerID, modelName, apiKey, base := resolveLLM(cfg, *provider, *model, *key, *baseURL)

	maxWorkers := *workers
	if maxWorkers <= 0 {
		maxWorkers = cfg.MaxWorkers
	}
	timeout := time.Duration(*timeoutSec) * time.Second
	if *timeoutSec <= 0 && cfg.RequestTimeoutSeconds > 0 {
		timeout = time.Duration(cfg.RequestTimeoutSeconds) * time.Second
	}
	outputDir := firstNonEmpty(*out, cfg.OutputDirectory)

	fmt.Printf("源文件   : %s (%s)\n", *file, translate.FileTypeName(*file))
	fmt.Printf("源语言   : %s\n", *from)
	fmt.Printf("目标语言 : %s\n", strings.Join(targetCodes, ", "))
	fmt.Printf("厂商/模型: %s / %s\n", providerID, fallback(modelName, "(默认)"))
	if *glossaryPath != "" {
		fmt.Printf("词表     : %s\n", *glossaryPath)
	}
	fmt.Println("开始翻译...")

	total := len(targetCodes)
	done := 0
	failed := 0
	results, runErr := translate.Run(context.Background(), translate.Options{
		SourceFile:     *file,
		TargetCodes:    targetCodes,
		SourceLanguage: *from,
		OutputDir:      outputDir,
		GlossaryPath:   *glossaryPath,
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

	fmt.Printf("\n完成：成功 %d，失败 %d，共 %d。\n", len(results)-failed, failed, len(results))
	if failed > 0 {
		os.Exit(1)
	}
}

// ─────────────────────────────────────────────
// 共享辅助
// ─────────────────────────────────────────────

func loadConfig() *config.AppConfig {
	cfg, err := config.Load()
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
}

// resolveLLM 按 命令行 > 环境变量 > config.json 的优先级解析厂商配置。
func resolveLLM(cfg *config.AppConfig, provider, model, key, baseURL string) (pid, modelName, apiKey, base string) {
	pid = provider
	if pid == "" {
		pid = cfg.LLM.ActiveProvider
	}
	pc := cfg.LLM.Providers[pid]
	apiKey = firstNonEmpty(key, os.Getenv("MLG_API_KEY"), pc.APIKey)
	modelName = firstNonEmpty(model, pc.Model)
	base = firstNonEmpty(baseURL, pc.BaseURL)
	return
}

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
