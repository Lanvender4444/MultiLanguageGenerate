package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/detector"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/filetype"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/language"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/processor"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/translator"
)

type App struct {
	fyneApp    fyne.App
	mainWindow fyne.Window
	cfg        *config.AppConfig

	sourcePanel   *SourcePanel
	langPanel     *LanguagePanel
	llmPanel      *LLMPanel
	progressPanel *ProgressPanel

	outputDirEntry *widget.Entry
	workerSlider   *widget.Slider
	workerLabel    *widget.Label
	timeoutEntry   *widget.Entry
	translateBtn   *widget.Button
	langFileBtn    *widget.Button

	mergeMode       *widget.RadioGroup
	mergeFormat     *widget.RadioGroup
	mergePromptEntry *widget.Entry

	cancelFunc context.CancelFunc
}

func NewApp(fyneApp fyne.App) *App {
	a := &App{fyneApp: fyneApp}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	SetThemeKind(ThemeKindFromString(cfg.Theme))
	fyneApp.Settings().SetTheme(NewCustomTheme())

	a.mainWindow = fyneApp.NewWindow("MultiLanguageGenerate")
	a.mainWindow.Resize(fyne.NewSize(800, 860))

	a.sourcePanel = NewSourcePanel(a.mainWindow)
	a.langPanel = NewLanguagePanel()
	a.llmPanel = NewLLMPanel(cfg, a.mainWindow)
	a.progressPanel = NewProgressPanel()

	a.outputDirEntry = widget.NewEntry()
	a.outputDirEntry.SetPlaceHolder("留空则与源文件同目录")
	a.outputDirEntry.SetText(cfg.OutputDirectory)

	a.workerSlider = widget.NewSlider(1, 20)
	a.workerSlider.Step = 1
	a.workerSlider.SetValue(float64(cfg.MaxWorkers))
	a.workerLabel = widget.NewLabel(fmt.Sprintf("%d", cfg.MaxWorkers))
	a.workerSlider.OnChanged = func(v float64) {
		a.workerLabel.SetText(fmt.Sprintf("%d", int(v)))
	}

	a.timeoutEntry = widget.NewEntry()
	a.timeoutEntry.SetText(fmt.Sprintf("%d", cfg.RequestTimeoutSeconds))

	a.mergeMode = widget.NewRadioGroup([]string{"不合并", "追加到源文件后面", "合并到新文件"}, nil)
	if cfg.MergeMode == "" {
		cfg.MergeMode = "none"
	}
	switch cfg.MergeMode {
	case "append":
		a.mergeMode.SetSelected("追加到源文件后面")
	case "newfile":
		a.mergeMode.SetSelected("合并到新文件")
	default:
		a.mergeMode.SetSelected("不合并")
	}

	a.mergeFormat = widget.NewRadioGroup([]string{"Markdown 表格", "纯文本", "自定义格式"}, nil)
	a.mergeFormat.Horizontal = true
	if cfg.MergeFormat == "" {
		cfg.MergeFormat = "markdown"
	}
	switch cfg.MergeFormat {
	case "plain":
		a.mergeFormat.SetSelected("纯文本")
	case "custom":
		a.mergeFormat.SetSelected("自定义格式")
	default:
		a.mergeFormat.SetSelected("Markdown 表格")
	}

	a.mergePromptEntry = widget.NewEntry()
	a.mergePromptEntry.SetPlaceHolder("输入自定义格式 Prompt...")
	a.mergePromptEntry.SetText(cfg.MergePrompt)

	a.mergeMode.OnChanged = func(v string) {
		isMerge := v != "不合并"
		if isMerge {
			a.outputDirEntry.Disable()
			a.outputDirEntry.Refresh()
			a.mergeFormat.Enable()
			a.mergeFormat.Refresh()
		} else {
			a.outputDirEntry.Enable()
			a.outputDirEntry.Refresh()
			a.mergeFormat.Disable()
			a.mergeFormat.Refresh()
			a.mergePromptEntry.Disable()
			a.mergePromptEntry.Refresh()
		}
	}
	a.mergeFormat.OnChanged = func(v string) {
		if v == "自定义格式" {
			a.mergePromptEntry.Enable()
		} else {
			a.mergePromptEntry.Disable()
		}
	}

	if a.mergeMode.Selected == "不合并" {
		a.mergeFormat.Disable()
		a.mergePromptEntry.Disable()
	} else {
		a.outputDirEntry.Disable()
		if a.mergeFormat.Selected != "自定义格式" {
			a.mergePromptEntry.Disable()
		}
	}

	a.translateBtn = widget.NewButton("开始翻译", a.startTranslation)
	a.translateBtn.Importance = widget.HighImportance

	a.langFileBtn = widget.NewButton("加载语言文件...", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			path := reader.URI().Path()
			reader.Close()
			a.cfg.LanguageFilePath = path
			a.loadLanguages()
		}, a.mainWindow)
	})

	a.loadLanguages()
	a.restoreLastSelected()

	a.langPanel.SetOnChange(func() {
		count := a.langPanel.SelectedCount()
		if count > 0 {
			a.translateBtn.SetText(fmt.Sprintf("开始翻译 (%d 种语言)", count))
		} else {
			a.translateBtn.SetText("开始翻译")
		}
	})

	a.mainWindow.SetContent(a.buildUI())
	return a
}

// ── loadLanguages / restoreLastSelected ─────────────────────────────────────

func (a *App) loadLanguages() {
	langPath := a.cfg.LanguageFilePath
	if langPath == "" {
		exePath, err := os.Executable()
		if err == nil {
			candidate := filepath.Join(filepath.Dir(exePath), "MultiLanguage.json")
			if _, err := os.Stat(candidate); err == nil {
				langPath = candidate
			}
		}
	}
	if langPath == "" {
		cwd, _ := os.Getwd()
		candidate := filepath.Join(cwd, "MultiLanguage.json")
		if _, err := os.Stat(candidate); err == nil {
			langPath = candidate
		}
	}
	if langPath != "" {
		languages, err := language.Load(langPath)
		if err == nil {
			a.langPanel.SetLanguages(languages)
			a.cfg.LanguageFilePath = langPath
			return
		}
	}
	languages, err := language.LoadEmbedded()
	if err != nil {
		dialog.ShowError(fmt.Errorf("无法加载语言列表: %v", err), a.mainWindow)
		return
	}
	a.langPanel.SetLanguages(languages)
}

func (a *App) restoreLastSelected() {
	if len(a.cfg.LastSelectedLanguages) > 0 {
		a.langPanel.SetSelectedCodes(a.cfg.LastSelectedLanguages)
	}
}

// ── UI builder ───────────────────────────────────────────────────────────────

func (a *App) buildUI() fyne.CanvasObject {
	tabs := container.NewAppTabs(
		container.NewTabItemWithIcon("翻译", theme.DocumentIcon(), a.buildTranslateTab()),
		container.NewTabItemWithIcon("模型", theme.SettingsIcon(), a.buildModelTab()),
		container.NewTabItemWithIcon("进度", theme.HistoryIcon(), a.buildProgressTab()),
	)
	tabs.SetTabLocation(container.TabLocationTop)

	header := NewAppHeader()
	header.OnToggleTheme = a.toggleTheme

	return container.NewBorder(header, nil, nil, nil, tabs)
}

// toggleTheme 在「木门·棕」与「水汽·银」之间切换并持久化。
func (a *App) toggleTheme() {
	if CurrentThemeKind() == ThemeWood {
		SetThemeKind(ThemeSilver)
	} else {
		SetThemeKind(ThemeWood)
	}
	a.cfg.Theme = CurrentThemeKind().String()
	a.cfg.Save()
	// 重新设主题触发全树刷新（自定义渲染器在 Refresh 中重读调色板）
	a.fyneApp.Settings().SetTheme(NewCustomTheme())
	a.mainWindow.Content().Refresh()
}

// gap returns a transparent fixed-height spacer between glass cards.
func gap(h float32) *FixedSpacer {
	return NewFixedSpacer(h)
}

// ── Tab: 翻译 ────────────────────────────────────────────────────────────────

func (a *App) buildTranslateTab() fyne.CanvasObject {
	sourceCard := NewGlassPanel("源文件", a.sourcePanel.Container())
	langCard := NewGlassPanel("目标语言", a.langPanel.Container())

	btnRow := container.NewVBox(a.langFileBtn, a.translateBtn)
	actionCard := NewGlassPanel("", btnRow)

	content := container.NewVBox(
		sourceCard,
		gap(10),
		langCard,
		gap(10),
		actionCard,
	)

	bg := NewWaterDropBg()
	return container.NewStack(bg, container.NewPadded(content))
}

// ── Tab: 模型 ────────────────────────────────────────────────────────────────

func (a *App) buildModelTab() fyne.CanvasObject {
	llmCard := NewGlassPanel("LLM 模型", a.llmPanel.Container())

	outputDirBtn := widget.NewButton("选择...", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			a.outputDirEntry.SetText(uri.Path())
		}, a.mainWindow)
	})

	optContent := container.NewVBox(
		widget.NewLabel("并发数"),
		container.NewBorder(nil, nil, nil, a.workerLabel, a.workerSlider),
		gap(4),
		widget.NewLabel("超时（秒）"),
		a.timeoutEntry,
	)
	optCard := NewGlassPanel("选项", optContent)

	mergeContent := container.NewVBox(
		widget.NewLabel("输出目录"),
		container.NewBorder(nil, nil, nil, outputDirBtn, a.outputDirEntry),
		gap(6),
		a.mergeMode,
		gap(4),
		widget.NewLabel("输出格式"),
		a.mergeFormat,
		a.mergePromptEntry,
	)
	mergeCard := NewGlassPanel("合并输出", mergeContent)

	content := container.NewVBox(
		llmCard,
		gap(10),
		optCard,
		gap(10),
		mergeCard,
	)

	bg := NewWaterDropBg()
	return container.NewStack(bg, container.NewPadded(container.NewVScroll(content)))
}

// ── Tab: 进度 ────────────────────────────────────────────────────────────────

func (a *App) buildProgressTab() fyne.CanvasObject {
	progressCard := NewGlassPanel("翻译进度", a.progressPanel.Container())

	bg := NewWaterDropBg()
	return container.NewStack(bg, container.NewPadded(container.NewVBox(progressCard)))
}

// ── startTranslation ─────────────────────────────────────────────────────────

func (a *App) startTranslation() {
	sourceContent, err := a.sourcePanel.ReadSourceContent()
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	if sourceContent == "" {
		dialog.ShowInformation("提示", "请先选择源文件", a.mainWindow)
		return
	}

	selectedLangs := a.langPanel.SelectedLanguages()
	if len(selectedLangs) == 0 {
		dialog.ShowInformation("提示", "请至少选择一种目标语言", a.mainWindow)
		return
	}

	pid, pc := a.llmPanel.GetCurrentProviderConfig()
	if pc.APIKey == "" && pid != "ollama" {
		dialog.ShowInformation("提示", "请配置 API Key", a.mainWindow)
		return
	}

	provider, err := llm.CreateProvider(pid, pc)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}

	sourceLang := a.sourcePanel.SourceLanguage
	if sourceLang == "auto" {
		_, name := detector.DetectLocal(sourceContent)
		sourceLang = name
	}

	outputDir := a.outputDirEntry.Text
	if outputDir == "" {
		outputDir = a.sourcePanel.SourceDir()
	}
	if outputDir == "" {
		outputDir, _ = os.Getwd()
	}

	mergeEnabled := a.mergeMode.Selected != "不合并"
	if mergeEnabled {
		outputDir = a.sourcePanel.SourceDir()
		if outputDir == "" {
			outputDir, _ = os.Getwd()
		}
	}

	maxWorkers := int(a.workerSlider.Value)
	timeoutSec := a.cfg.RequestTimeoutSeconds
	fmt.Sscanf(a.timeoutEntry.Text, "%d", &timeoutSec)

	srcFileType := filetype.DetectFile(a.sourcePanel.SourceFile)
	ext := filepath.Ext(a.sourcePanel.SourceFile)
	typeInfo := filetype.TypeInfoOf(srcFileType)
	a.progressPanel.AddStatusLine(fmt.Sprintf("文件类型: %s (%s)", typeInfo.Description, ext))

	engine := translator.NewEngine(provider, pc.Model, maxWorkers, time.Duration(timeoutSec)*time.Second)

	jobs := make([]translator.Job, 0, len(selectedLangs))
	codes := make([]string, 0, len(selectedLangs))
	for _, lang := range selectedLangs {
		jobs = append(jobs, translator.Job{
			SourceText:     sourceContent,
			SourceFile:     a.sourcePanel.SourceFile,
			SourceLanguage: sourceLang,
			TargetCode:     lang.Code,
			TargetName:     lang.Name,
			OutputDir:      outputDir,
			SourceFileType: srcFileType,
			SkipOutput:     mergeEnabled,
		})
		codes = append(codes, lang.Code)
	}

	a.progressPanel.InitJobs(codes)

	a.cfg.LastSelectedLanguages = a.langPanel.GetSelectedCodes()
	a.cfg.OutputDirectory = outputDir
	a.cfg.MaxWorkers = maxWorkers
	a.cfg.RequestTimeoutSeconds = timeoutSec
	switch a.mergeMode.Selected {
	case "追加到源文件后面":
		a.cfg.MergeMode = "append"
	case "合并到新文件":
		a.cfg.MergeMode = "newfile"
	default:
		a.cfg.MergeMode = "none"
	}
	switch a.mergeFormat.Selected {
	case "纯文本":
		a.cfg.MergeFormat = "plain"
	case "自定义格式":
		a.cfg.MergeFormat = "custom"
	default:
		a.cfg.MergeFormat = "markdown"
	}
	a.cfg.MergePrompt = a.mergePromptEntry.Text
	a.cfg.Save()

	ctx, cancel := context.WithCancel(context.Background())
	a.cancelFunc = cancel

	a.translateBtn.SetText("取消翻译")
	a.translateBtn.Importance = widget.DangerImportance
	a.translateBtn.OnTapped = func() {
		cancel()
		a.translateBtn.SetText("开始翻译")
		a.translateBtn.Importance = widget.HighImportance
		a.translateBtn.OnTapped = a.startTranslation
		a.translateBtn.Refresh()
	}
	a.translateBtn.Refresh()

	progress := make(chan translator.Result, len(jobs))
	for _, code := range codes {
		a.progressPanel.SetTranslating(code)
	}

	go func() { engine.Run(ctx, jobs, progress) }()

	go func() {
		var results []translator.Result
		for result := range progress {
			r := result
			results = append(results, r)
			fyne.Do(func() { a.progressPanel.UpdateResult(r) })
		}
		fyne.Do(func() {
			a.translateBtn.SetText(fmt.Sprintf("开始翻译 (%d 种语言)", len(selectedLangs)))
			a.translateBtn.Importance = widget.HighImportance
			a.translateBtn.OnTapped = a.startTranslation
			a.translateBtn.Refresh()

			if a.mergeMode.Selected != "不合并" && len(results) > 0 {
				a.progressPanel.AddStatusLine("正在合并...")
				go a.doMerge(results, selectedLangs, sourceLang, srcFileType, provider, pc.Model, outputDir, sourceContent)
			}
		})
	}()
}

func (a *App) Run() {
	a.mainWindow.ShowAndRun()
}

func (a *App) doMerge(results []translator.Result, langs []language.Language, sourceLang string, srcFileType filetype.FileType, provider llm.Provider, model string, outputDir string, sourceText string) {
	sourceLabel := fmt.Sprintf("原文 (%s)", sourceLang)
	var langTexts []string
	langTexts = append(langTexts, fmt.Sprintf("=== %s ===\n%s", sourceLabel, sourceText))

	for _, r := range results {
		if r.Error != nil {
			continue
		}
		label := r.TargetCode
		for _, l := range langs {
			if l.Code == r.TargetCode {
				label = l.Name + " (" + l.Code + ")"
				break
			}
		}
		langTexts = append(langTexts, fmt.Sprintf("=== %s ===\n%s", label, r.TranslatedText))
	}

	if len(langTexts) < 2 {
		fyne.Do(func() { a.progressPanel.AddStatusLine("合并失败：无有效翻译结果") })
		return
	}

	mergeFormat := a.mergeFormat.Selected
	var mergePrompt, defaultExt string
	switch mergeFormat {
	case "纯文本":
		defaultExt = ".txt"
		mergePrompt = "请将上述原文及多种语言翻译整理成一个纯文本对照文档。格式：每段先显示语言名称，然后是该语言的文本内容，最后接一个空行。只输出最终文档，不要解释。"
	case "自定义格式":
		defaultExt = ".txt"
		mergePrompt = strings.TrimSpace(a.mergePromptEntry.Text)
		if mergePrompt == "" {
			mergePrompt = "请将上述原文及多种语言翻译整理成一个纯文本多语言对照文档。使用“语言名：内容”的格式，每段之间空一行。只输出最终文档，不要解释。"
		}
	default:
		defaultExt = ".md"
		mergePrompt = "请将上述原文及多种语言翻译整理成一个 Markdown 表格，第一列是语言名，第二列是内容。表格放在文档最前面。保留所有原文格式。只输出最终 Markdown 文档，不要解释。"
	}

	fullPrompt := fmt.Sprintf("以下是将一份源文档（源语言：%s）翻译成多种语言的结果，附原文：\n\n%s\n\n---\n%s",
		sourceLang, strings.Join(langTexts, "\n\n"), mergePrompt)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(a.cfg.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	merged, err := provider.Translate(ctx, llm.TranslateRequest{
		SourceText:     fullPrompt,
		SourceLanguage: sourceLang,
		TargetLanguage: "multi",
		TargetCode:     "xx",
		Model:          model,
		SourceType:     filetype.FileTypePlainText,
		SystemPrompt:   "You are a document formatting assistant. Combine the source text and multiple translations into a single well-formatted document. Output ONLY the document, no explanations or preamble.",
	})
	if err != nil {
		fyne.Do(func() {
			a.progressPanel.AddStatusLine(fmt.Sprintf("合并失败: %v", err))
			dialog.ShowError(err, a.mainWindow)
		})
		return
	}

	merged = processor.StripCodeFence(merged)

	srcName := filepath.Base(a.sourcePanel.SourceFile)
	ext := filepath.Ext(srcName)
	base := srcName[:len(srcName)-len(ext)]

	var mergePath string
	mode := a.mergeMode.Selected
	if mode == "追加到源文件后面" {
		if srcFileType == filetype.FileTypePlainText || srcFileType == filetype.FileTypeMarkdown ||
			srcFileType == filetype.FileTypeSRT || srcFileType == filetype.FileTypePO ||
			srcFileType == filetype.FileTypeHTML || srcFileType == filetype.FileTypeXML ||
			srcFileType == filetype.FileTypeJSON || srcFileType == filetype.FileTypeCSV {
			mergePath = filepath.Join(outputDir, srcName)
		} else {
			mergePath = filepath.Join(outputDir, base+defaultExt)
		}
	} else {
		mergePath = filepath.Join(outputDir, base+"_MultiLanguageGenerate"+defaultExt)
	}

	if err := os.WriteFile(mergePath, []byte(merged), 0644); err != nil {
		fyne.Do(func() {
			dialog.ShowError(err, a.mainWindow)
		})
		return
	}

	fyne.Do(func() {
		a.progressPanel.AddStatusLine(fmt.Sprintf("合并完成: %s", filepath.Base(mergePath)))
		dialog.ShowInformation("合并完成", fmt.Sprintf("多语言合并文件已保存到：\n%s", mergePath), a.mainWindow)
	})
}
