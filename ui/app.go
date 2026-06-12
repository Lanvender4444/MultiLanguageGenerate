package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

	a.mainWindow = fyneApp.NewWindow("MultiLanguage")
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
		widget.NewLabel("输出目录"),
		container.NewBorder(nil, nil, nil, outputDirBtn, a.outputDirEntry),
		gap(4),
		widget.NewLabel("并发数"),
		container.NewBorder(nil, nil, nil, a.workerLabel, a.workerSlider),
		gap(4),
		widget.NewLabel("超时（秒）"),
		a.timeoutEntry,
	)
	optCard := NewGlassPanel("选项", optContent)

	content := container.NewVBox(
		llmCard,
		gap(10),
		optCard,
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
		})
		codes = append(codes, lang.Code)
	}

	a.progressPanel.InitJobs(codes)

	a.cfg.LastSelectedLanguages = a.langPanel.GetSelectedCodes()
	a.cfg.OutputDirectory = outputDir
	a.cfg.MaxWorkers = maxWorkers
	a.cfg.RequestTimeoutSeconds = timeoutSec
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
		for result := range progress {
			r := result
			fyne.Do(func() { a.progressPanel.UpdateResult(r) })
		}
		fyne.Do(func() {
			a.translateBtn.SetText(fmt.Sprintf("开始翻译 (%d 种语言)", len(selectedLangs)))
			a.translateBtn.Importance = widget.HighImportance
			a.translateBtn.OnTapped = a.startTranslation
			a.translateBtn.Refresh()
		})
	}()
}

func (a *App) Run() {
	a.mainWindow.ShowAndRun()
}
