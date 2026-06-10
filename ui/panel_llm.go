package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/yourname/MultiLanguageGenerate/internal/config"
	"github.com/yourname/MultiLanguageGenerate/internal/llm"
)

type LLMPanel struct {
	widget.BaseWidget
	cfg    *config.AppConfig
	window fyne.Window

	providerSelect *widget.Select
	modelSelect    *widget.Select
	apiKeyEntry    *widget.Entry
	customURLEntry *widget.Entry
	saveBtn        *widget.Button
	refreshBtn     *widget.Button

	OnProviderChanged func(providerID string)
}

func NewLLMPanel(cfg *config.AppConfig, window fyne.Window) *LLMPanel {
	p := &LLMPanel{
		cfg:    cfg,
		window: window,
	}

	providers := llm.AllProviders()
	providerNames := make([]string, 0, len(providers))
	for _, info := range providers {
		providerNames = append(providerNames, info.DisplayName)
	}

	displayToID := make(map[string]string)
	idToDisplay := make(map[string]string)
	for _, info := range providers {
		displayToID[info.DisplayName] = info.ID
		idToDisplay[info.ID] = info.DisplayName
	}

	p.providerSelect = widget.NewSelect(providerNames, func(s string) {
		pid := displayToID[s]
		p.cfg.LLM.ActiveProvider = pid
		if pc, ok := p.cfg.LLM.Providers[pid]; ok {
			p.apiKeyEntry.SetText(pc.APIKey)
			p.modelSelect.SetSelected(pc.Model)
			info, _ := llm.GetProviderInfo(pid)
			if info.BaseURL != "" && pc.BaseURL == "" {
				p.customURLEntry.SetText("")
			} else {
				p.customURLEntry.SetText(pc.BaseURL)
			}
		}
		if p.OnProviderChanged != nil {
			p.OnProviderChanged(pid)
		}
	})

	p.modelSelect = widget.NewSelect([]string{}, func(s string) {
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.Model = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	})

	p.apiKeyEntry = widget.NewPasswordEntry()
	p.apiKeyEntry.SetPlaceHolder("输入 API Key...")
	p.apiKeyEntry.OnChanged = func(s string) {
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.APIKey = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	}

	p.customURLEntry = widget.NewEntry()
	p.customURLEntry.SetPlaceHolder("留空使用默认 URL...")
	p.customURLEntry.OnChanged = func(s string) {
		if pc, ok := p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider]; ok {
			pc.BaseURL = s
			p.cfg.LLM.Providers[p.cfg.LLM.ActiveProvider] = pc
		}
	}

	p.refreshBtn = widget.NewButton("刷新模型列表", func() {
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
	})

	p.saveBtn = widget.NewButton("保存配置", func() {
		if err := p.cfg.Save(); err != nil {
			dialog.ShowError(err, p.window)
		} else {
			dialog.ShowInformation("保存成功", "配置已保存", p.window)
		}
	})

	if activeDisplay, ok := idToDisplay[cfg.LLM.ActiveProvider]; ok {
		p.providerSelect.SetSelected(activeDisplay)
	}

	return p
}

func (p *LLMPanel) CreateRenderer() fyne.WidgetRenderer {
	box := container.NewVBox(
		widget.NewLabelWithStyle("🤖 LLM 配置", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewGridWithColumns(2,
			widget.NewLabel("Provider:"),
			p.providerSelect,
			widget.NewLabel("Model:"),
			container.NewBorder(nil, nil, nil, p.refreshBtn, p.modelSelect),
			widget.NewLabel("API Key:"),
			p.apiKeyEntry,
			widget.NewLabel("Custom URL:"),
			p.customURLEntry,
		),
		p.saveBtn,
	)
	return widget.NewSimpleRenderer(box)
}

func (p *LLMPanel) GetCurrentProviderConfig() (string, config.ProviderConfig) {
	pid := p.cfg.LLM.ActiveProvider
	pc := p.cfg.LLM.Providers[pid]
	return pid, pc
}
