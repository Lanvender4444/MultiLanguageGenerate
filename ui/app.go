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
	"fyne.io/fyne/v2/widget"

	"github.com/yourname/MultiLanguageGenerate/internal/config"
	"github.com/yourname/MultiLanguageGenerate/internal/detector"
	"github.com/yourname/MultiLanguageGenerate/internal/language"
	"github.com/yourname/MultiLanguageGenerate/internal/llm"
	"github.com/yourname/MultiLanguageGenerate/internal/translator"
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
	timeoutEntry   *widget.Entry
	translateBtn   *widget.Button

	cancelFunc context.CancelFunc
}

func NewApp(fyneApp fyne.App) *App {
	a := &App{
		fyneApp: fyneApp,
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}
	a.cfg = cfg

	a.mainWindow = fyneApp.NewWindow("MultiLanguageGenerate")
	a.mainWindow.Resize(fyne.NewSize(800, 700))

	a.sourcePanel = NewSourcePanel(a.mainWindow)
	a.langPanel = NewLanguagePanel()
	a.llmPanel = NewLLMPanel(cfg, a.mainWindow)
	a.progressPanel = NewProgressPanel()

	a.loadLanguages()

	a.outputDirEntry = widget.NewEntry()
	a.outputDirEntry.SetPlaceHolder("同源文件目录")
	a.outputDirEntry.SetText(cfg.OutputDirectory)

	a.workerSlider = widget.NewSlider(1, 20)
	a.workerSlider.Step = 1
	a.workerSlider.SetValue(float64(cfg.MaxWorkers))

	a.timeoutEntry = widget.NewEntry()
	a.timeoutEntry.SetText(fmt.Sprintf("%d", cfg.RequestTimeoutSeconds))

	a.translateBtn = widget.NewButton("🚀 开始翻译", a.startTranslation)

	a.langPanel.SetOnChange(func() {
		count := a.langPanel.SelectedCount()
		if count > 0 {
			a.translateBtn.SetText(fmt.Sprintf("🚀 开始翻译 (%d 种语言)", count))
		} else {
			a.translateBtn.SetText("🚀 开始翻译")
		}
	})

	a.restoreLastSelected()

	a.buildUI()

	return a
}

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
		}
	}
}

func (a *App) restoreLastSelected() {
	if len(a.cfg.LastSelectedLanguages) > 0 {
		a.langPanel.SetSelectedCodes(a.cfg.LastSelectedLanguages)
	}
}

func (a *App) buildUI() {
	outputDirBtn := widget.NewButton("选择", func() {
		dialog.ShowFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			a.outputDirEntry.SetText(uri.Path())
		}, a.mainWindow)
	})

	optionsPanel := container.NewVBox(
		widget.NewLabelWithStyle("⚙️ 选项", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			widget.NewLabel("输出目录:"),
			container.NewBorder(nil, nil, nil, outputDirBtn, a.outputDirEntry),
			widget.NewLabel("并发数:"),
			container.NewHBox(a.workerSlider, widget.NewLabel("workers")),
			widget.NewLabel("超时:"),
			container.NewHBox(a.timeoutEntry, widget.NewLabel("秒")),
		),
	)

	langFileBtn := widget.NewButton("选择语言文件", func() {
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

	content := container.NewVBox(
		a.sourcePanel,
		widget.NewSeparator(),
		a.langPanel,
		widget.NewSeparator(),
		a.llmPanel,
		widget.NewSeparator(),
		optionsPanel,
		widget.NewSeparator(),
		langFileBtn,
		a.translateBtn,
		widget.NewSeparator(),
		a.progressPanel,
	)

	scroll := container.NewVScroll(content)
	scroll.SetMinSize(fyne.NewSize(780, 680))

	a.mainWindow.SetContent(scroll)
}

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
		code, name := detector.DetectLocal(sourceContent)
		sourceLang = name
		_ = code
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

	a.translateBtn.SetText("⏹ 取消翻译")
	a.translateBtn.OnTapped = func() {
		cancel()
		a.translateBtn.SetText("🚀 开始翻译")
		a.translateBtn.OnTapped = a.startTranslation
	}

	progress := make(chan translator.Result, len(jobs))

	go func() {
		engine.Run(ctx, jobs, progress)
	}()

	go func() {
		for result := range progress {
			r := result
			a.progressPanel.UpdateResult(r)
		}
		a.translateBtn.SetText(fmt.Sprintf("🚀 开始翻译 (%d 种语言)", len(selectedLangs)))
		a.translateBtn.OnTapped = a.startTranslation
		a.translateBtn.Refresh()
	}()
}

func (a *App) Run() {
	a.mainWindow.ShowAndRun()
}
