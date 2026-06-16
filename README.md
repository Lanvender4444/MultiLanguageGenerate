# MultiLanguageGenerate
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.22+-00ADD8.svg?style=flat&logo=go)](https://golang.org)
![Cgo Supported](https://img.shields.io/badge/Cgo-Supported-blue?style=flat&logo=go&logoColor=white)
![GCC Required](https://img.shields.io/badge/GCC-Required-51789c?style=flat&logo=gnu-compiler-collection&logoColor=white)
![Fyne GUI](https://img.shields.io/badge/Fyne-GUI_Framework-00d2ff?style=flat&logo=go&logoColor=white)



MultiLanguageGenerate 是一款基于 Golang 的，高并发 多语言 多格式 翻译文件生成器。

## 核心特性
**高并发**: 利用Golang的包并发特性开发。

**多类型**: 匹配多种文件类型，包括docx、xlsx等，更灵活更省心的进行翻译。

**AI驱动**: 利用 LLM API，进行翻译。

本项目提供三种使用方式：**桌面 GUI**、**命令行 CLI**、以及作为 **Go 库**被其它程序引用。

## 快速上手

### 一、桌面 GUI（需要 CGO / GCC / OpenGL）

```powershell
# for Windows
./build.ps1
```

下载发行版：[👉 GitHub Releases](https://github.com/Lanvender4444/MultiLanguageGenerate/releases)

### 二、命令行 CLI（纯 Go，无需 CGO）

编译：

```powershell
# Windows
./build-cli.ps1
```

```bash
# Linux / macOS
./build-cli.sh
# 或直接：
CGO_ENABLED=0 go build -o mlg-cli ./cmd/mlg-cli
```

使用：

```bash
# 列出全部可用语言 / 厂商
mlg-cli -list-langs
mlg-cli -list-providers

# 把 report.docx 翻译成英语和日语（复用 GUI 已保存的 config.json 中的厂商与 Key）
mlg-cli -file report.docx -to en,ja

# 临时指定厂商、模型与 Key（覆盖配置）
mlg-cli -file notes.md -to fr,de -provider deepseek -model deepseek-chat -key sk-xxx

# 指定输出目录、并发数、超时
mlg-cli -file book.epub -to zh-CN,ko -out ./dist -workers 8 -timeout 180
```

CLI 配置来源优先级：**命令行参数 > 环境变量 `MLG_API_KEY`（仅 Key）> config.json**。
config.json 与 GUI 共用（位于系统用户配置目录下的 `MultiLanguageGenerate/config.json`）。

常用参数：

| 参数 | 说明 |
|------|------|
| `-file` | 源文件路径（必填） |
| `-to` | 目标语言代码，逗号分隔，如 `en,ja,fr`（必填） |
| `-from` | 源语言显示名；`auto`=自动识别（默认） |
| `-out` | 输出目录；留空则与源文件同目录 |
| `-provider` | 厂商 ID；留空用配置中的 active_provider |
| `-model` | 模型名 |
| `-key` | API Key |
| `-base-url` | 自定义服务地址 |
| `-workers` | 并发数 |
| `-timeout` | 单请求超时(秒) |
| `-list-langs` / `-list-providers` | 打印清单后退出 |

### 三、作为 Go 库引用

```bash
go get github.com/Lanvender4444/MultiLanguageGenerate/translate
```

```go
package main

import (
    "context"
    "fmt"

    "github.com/Lanvender4444/MultiLanguageGenerate/translate"
)

func main() {
    results, err := translate.Run(context.Background(), translate.Options{
        SourceFile:  "report.docx",
        TargetCodes: []string{"en", "ja"},
        Provider:    "deepseek",
        Model:       "deepseek-chat",
        APIKey:      "sk-xxx",
        // OutputDir 留空则与源文件同目录；
        // 可选 OnResult 回调用于流式进度。
    })
    if err != nil {
        panic(err) // 仅前置错误（参数缺失/厂商未知等）
    }
    for _, r := range results {
        if r.Err != nil {
            fmt.Printf("✗ %s 失败: %v\n", r.TargetCode, r.Err)
        } else {
            fmt.Printf("✓ %s → %s\n", r.TargetCode, r.OutputPath)
        }
    }
}
```

`translate` 包只依赖纯 Go 的内部实现，不引入 Fyne/CGO，可在服务器、CI 等无图形环境中使用。
辅助函数：`translate.Providers()` 列厂商、`translate.Languages()` 列语言、`translate.FileTypeName(path)` 识别文件类型。

