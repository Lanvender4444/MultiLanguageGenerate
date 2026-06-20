package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/translate"
)

// runGlossary 处理 `mlg-cli glossary <gen|show>`
func runGlossary(argv []string) {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "用法：mlg-cli glossary <gen|show> [选项]")
		os.Exit(2)
	}
	switch argv[0] {
	case "gen", "generate":
		glossaryGen(argv[1:])
	case "show", "list":
		glossaryShow(argv[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知 glossary 子命令：%q（可用：gen/show）\n", argv[0])
		os.Exit(2)
	}
}

func glossaryGen(argv []string) {
	fs := flag.NewFlagSet("glossary gen", flag.ExitOnError)
	prompt := fs.String("prompt", "", "提示词：描述要构造的专业名词/人物/术语（必填）")
	to := fs.String("to", "", "目标语言代码，逗号分隔（必填）")
	from := fs.String("from", "", "源语言显示名（可选）")
	out := fs.String("o", "", "输出词表 JSON 路径（可选；不填则打印到屏幕）")
	provider := fs.String("provider", "", "厂商 ID")
	model := fs.String("model", "", "模型名")
	key := fs.String("key", "", "API Key")
	baseURL := fs.String("base-url", "", "自定义服务地址")
	kbDir := fs.String("kb-dir", "", "知识库目录；留空用默认")
	sel := fs.String("select", "", "勾选的知识库文档名或标签，逗号分隔；空=全部")
	topk := fs.Int("topk", 8, "RAG 检索片段数")
	embedModel := fs.String("embed-model", "", "embedding 模型名；提供则用向量检索")
	_ = fs.Parse(argv)

	if *prompt == "" || *to == "" {
		fmt.Fprintln(os.Stderr, "错误：-prompt 和 -to 为必填参数。")
		fmt.Fprintln(os.Stderr, `示例：mlg-cli glossary gen -prompt "提取人名与技术术语，风格正式" -to en,ja -o glossary.json`)
		os.Exit(2)
	}

	cfg := loadConfig()
	pid, modelName, apiKey, base := resolveLLM(cfg, *provider, *model, *key, *baseURL)
	em := firstNonEmpty(*embedModel, os.Getenv("MLG_EMBED_MODEL"))

	fmt.Printf("厂商/模型 : %s / %s\n", pid, fallback(modelName, "(默认)"))
	fmt.Printf("目标语言  : %s\n", *to)
	fmt.Println("正在依据知识库检索 + AI 生成专业名词词表...")

	doc, err := translate.GenerateGlossary(context.Background(), translate.GlossaryGenOptions{
		Prompt:         *prompt,
		TargetCodes:    splitCSV(*to),
		SourceLanguage: *from,
		Provider:       pid,
		Model:          modelName,
		APIKey:         apiKey,
		BaseURL:        base,
		KBDir:          *kbDir,
		SelectKB:       splitCSV(*sel),
		TopK:           *topk,
		EmbedModel:     em,
		OutputPath:     *out,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n生成完成：%d 个条目", len(doc.Entries))
	if *out != "" {
		fmt.Printf("，已写入 %s", *out)
	}
	fmt.Println()
	previewGlossary(doc)
}

func glossaryShow(argv []string) {
	fs := flag.NewFlagSet("glossary show", flag.ExitOnError)
	f := fs.String("f", "", "词表 JSON 路径（必填）")
	_ = fs.Parse(argv)
	if *f == "" {
		fmt.Fprintln(os.Stderr, "错误：-f 词表路径为必填。")
		os.Exit(2)
	}
	data, err := os.ReadFile(*f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取失败：%v\n", err)
		os.Exit(1)
	}
	var doc translate.GlossaryDoc
	if err := json.Unmarshal(data, &doc); err != nil {
		fmt.Fprintf(os.Stderr, "解析失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("词表：%s（%d 个条目）\n", fallback(doc.Name, "(未命名)"), len(doc.Entries))
	previewGlossary(&doc)
}

func previewGlossary(doc *translate.GlossaryDoc) {
	max := len(doc.Entries)
	if max > 20 {
		max = 20
	}
	for _, e := range doc.Entries[:max] {
		var trans []string
		for code, t := range e.Translation {
			trans = append(trans, code+"="+t)
		}
		fmt.Printf("  • %-20s [%s] %s\n", e.Term, e.Type, strings.Join(trans, "  "))
	}
	if len(doc.Entries) > max {
		fmt.Printf("  ...（其余 %d 条见文件）\n", len(doc.Entries)-max)
	}
}
