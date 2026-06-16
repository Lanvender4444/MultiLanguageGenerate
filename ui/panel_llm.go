package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/config"
	"github.com/Lanvender4444/MultiLanguageGenerate/internal/llm"
)

type LLMPanel struct {
	cfg     *config.AppConfig
	window  fyne.Window
	maps    displayToIDMap

	providerSelect *widget.Select
	modelSelect    *widget.Select
	apiKeyEntry    *widget.Entry
	customURLEntry *widget.Entry

	box     *fyne.Container
	syncing bool
}

type displayToIDMap struct {
	displayToID map[string]string
	idToDisplay map[string]string
	names       []string
}

func NewLLMPanel(cfg *config.AppConfig, window fyne.Window) *LLMPanel {
	p := &LLMPanel{cfg: cfg, window: window}
	p.maps.displayToID = make(map[string]string)
	p.maps.idToDisplay = make(map[string]string)

	for _, info := range llm.AllProviders() {
		p.maps.displayToID[info.DisplayName] = info.ID
		p.maps.idToDisplay[info.ID] = info.DisplayName
		p.maps.names = append(p.maps.names, info.DisplayName)
	}

	p.providerSelect = widget.NewSelect(p.maps.names, func(s string) {
		if p.syncing {
			return
		}
		p.cfg.LLM.ActiveProvider = p.maps.displayToID[s]
		p.syncFieldsFromConfig()
	})

	p.modelSelect = widget.NewSelect([]string{"(点击刷新获取列表)"}, func(s string) {
		if p.syncing {
			return
		}
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.Model = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	})

	p.apiKeyEntry = widget.NewPasswordEntry()
	p.apiKeyEntry.SetPlaceHolder("输入 API Key…")
	p.apiKeyEntry.OnChanged = func(s string) {
		if p.syncing {
			return
		}
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.APIKey = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	}

	p.customURLEntry = widget.NewEntry()
	p.customURLEntry.SetPlaceHolder("留空使用默认 URL…")
	p.customURLEntry.OnChanged = func(s string) {
		if p.syncing {
			return
		}
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.BaseURL = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	}

	p.syncFieldsFromConfig()
	return p
}

func (p *LLMPanel) syncFieldsFromConfig() {
	p.syncing = true
	defer func() { p.syncing = false }()

	pid := p.cfg.LLM.ActiveProvider
	if display, ok := p.maps.idToDisplay[pid]; ok {
		p.providerSelect.SetSelected(display)
	}
	if pc, ok := p.cfg.LLM.Providers[pid]; ok {
		p.apiKeyEntry.SetText(pc.APIKey)
		p.customURLEntry.SetText(pc.BaseURL)
	}
}

func (p *LLMPanel) Container() *fyne.Container {
	refreshBtn := widget.NewButton("刷新", p.refreshModels)

	saveBtn := widget.NewButton("保存配置", func() {
		if err := p.cfg.Save(); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("提示", "配置已保存", p.window)
		}
	})
	saveBtn.Importance = widget.MediumImportance

	form := container.NewVBox(
		widget.NewLabel("Provider"),
		p.providerSelect,
		widget.NewLabel("模型"),
		container.NewBorder(nil, nil, nil, refreshBtn, p.modelSelect),
		widget.NewLabel("API Key"),
		p.apiKeyEntry,
		widget.NewLabel("自定义 URL"),
		p.customURLEntry,
		saveBtn,
	)

	p.box = form
	return p.box
}

func (p *LLMPanel) refreshModels() {
	pid := p.cfg.LLM.ActiveProvider
	pc, ok := p.cfg.LLM.Providers[pid]
	if !ok {
		return
	}
	provider, err := llm.CreateProvider(pid, pc)
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}
	models, err := provider.ListModels(context.Background())
	if err != nil {
		dialog.ShowError(err, p.window)
		return
	}
	p.modelSelect.Options = models
	if len(models) > 0 {
		p.modelSelect.SetSelected(models[0])
	}
	p.modelSelect.Refresh()
}

func (p *LLMPanel) GetCurrentProviderConfig() (string, config.ProviderConfig) {
	pid := p.cfg.LLM.ActiveProvider
	return pid, p.cfg.LLM.Providers[pid]
}
