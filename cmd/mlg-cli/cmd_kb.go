package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Lanvender4444/MultiLanguageGenerate/translate"
)

// runKB 处理 `mlg-cli kb <add|list|index|search>`
func runKB(argv []string) {
	if len(argv) == 0 {
		fmt.Fprintln(os.Stderr, "用法：mlg-cli kb <add|list|index|search> [选项]")
		os.Exit(2)
	}
	sub := argv[0]
	rest := argv[1:]

	switch sub {
	case "add":
		kbAdd(rest)
	case "list", "ls":
		kbList(rest)
	case "index", "reindex":
		kbIndex(rest)
	case "search", "query":
		kbSearch(rest)
	default:
		fmt.Fprintf(os.Stderr, "未知 kb 子命令：%q（可用：add/list/index/search）\n", sub)
		os.Exit(2)
	}
}

func kbAdd(argv []string) {
	fs := flag.NewFlagSet("kb add", flag.ExitOnError)
	dir := fs.String("dir", "", "知识库目录；留空用默认")
	path := fs.String("path", "", "待加入的文件路径（也可作为第一个位置参数）")
	link := fs.String("link", "copy", "加入方式：copy | soft | hard")
	tags := fs.String("tags", "", "标签，逗号分隔，如 style,legal")
	_ = fs.Parse(argv)

	src := *path
	if src == "" && fs.NArg() > 0 {
		src = fs.Arg(0)
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "错误：需指定文件路径（-path 或位置参数）。")
		os.Exit(2)
	}

	if err := translate.KBAdd(translate.KBAddOptions{
		Dir:      *dir,
		Path:     src,
		LinkMode: *link,
		Tags:     splitCSV(*tags),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "加入失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("已加入知识库：%s（方式=%s，标签=%s）\n", src, *link, fallback(*tags, "无"))
	fmt.Println("索引已更新（词法）。如需向量检索，请运行： mlg-cli kb index -embed-model <模型>")
}

func kbList(argv []string) {
	fs := flag.NewFlagSet("kb list", flag.ExitOnError)
	dir := fs.String("dir", "", "知识库目录；留空用默认")
	_ = fs.Parse(argv)

	docs, err := translate.KBList(*dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "读取失败：%v\n", err)
		os.Exit(1)
	}
	if len(docs) == 0 {
		fmt.Println("知识库为空。用 mlg-cli kb add <文件> 添加内容。")
		return
	}
	fmt.Printf("知识库文档（共 %d 个）：\n", len(docs))
	for _, d := range docs {
		tag := ""
		if len(d.Tags) > 0 {
			tag = "  [" + strings.Join(d.Tags, ", ") + "]"
		}
		fmt.Printf("  %s%s\n", d.Name, tag)
	}
}

func kbIndex(argv []string) {
	fs := flag.NewFlagSet("kb index", flag.ExitOnError)
	dir := fs.String("dir", "", "知识库目录；留空用默认")
	provider := fs.String("provider", "", "厂商 ID（用于 embedding）")
	key := fs.String("key", "", "API Key")
	baseURL := fs.String("base-url", "", "自定义服务地址")
	embedModel := fs.String("embed-model", "", "embedding 模型名；留空则只建词法索引")
	_ = fs.Parse(argv)

	cfg := loadConfig()
	pid, _, apiKey, base := resolveLLM(cfg, *provider, "", *key, *baseURL)
	em := firstNonEmpty(*embedModel, os.Getenv("MLG_EMBED_MODEL"))

	mode := "词法(BM25)"
	if em != "" {
		mode = "向量(" + em + ") + 词法回退"
	}
	fmt.Printf("正在重建索引（%s）...\n", mode)

	if err := translate.KBIndex(context.Background(), translate.KBIndexOptions{
		Dir:        *dir,
		Provider:   pid,
		APIKey:     apiKey,
		BaseURL:    base,
		EmbedModel: em,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "建索引失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Println("索引完成。")
}

func kbSearch(argv []string) {
	fs := flag.NewFlagSet("kb search", flag.ExitOnError)
	dir := fs.String("dir", "", "知识库目录；留空用默认")
	q := fs.String("q", "", "查询文本（必填）")
	k := fs.Int("k", 5, "返回片段数")
	sel := fs.String("select", "", "限定文档名或标签，逗号分隔；空=全部")
	provider := fs.String("provider", "", "厂商 ID（向量检索用）")
	key := fs.String("key", "", "API Key")
	baseURL := fs.String("base-url", "", "自定义服务地址")
	embedModel := fs.String("embed-model", "", "embedding 模型名；提供则用向量检索")
	_ = fs.Parse(argv)

	if *q == "" {
		fmt.Fprintln(os.Stderr, "错误：-q 查询文本为必填。")
		os.Exit(2)
	}

	cfg := loadConfig()
	pid, _, apiKey, base := resolveLLM(cfg, *provider, "", *key, *baseURL)
	em := firstNonEmpty(*embedModel, os.Getenv("MLG_EMBED_MODEL"))

	hits, err := translate.KBSearch(context.Background(), translate.KBSearchOptions{
		Dir:        *dir,
		Query:      *q,
		TopK:       *k,
		Selected:   splitCSV(*sel),
		Provider:   pid,
		APIKey:     apiKey,
		BaseURL:    base,
		EmbedModel: em,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "检索失败：%v\n", err)
		os.Exit(1)
	}
	if len(hits) == 0 {
		fmt.Println("无匹配结果。")
		return
	}
	fmt.Printf("命中 %d 条：\n", len(hits))
	for i, h := range hits {
		fmt.Printf("\n[%d] doc=%s score=%.4f\n%s\n", i+1, h.Doc, h.Score, truncate(h.Text, 300))
	}
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
