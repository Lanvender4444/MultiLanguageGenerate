package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/yourname/MultiLanguageGenerate/internal/language"
)

type LanguagePanel struct {
	widget.BaseWidget
	languages []language.Language
	checks    map[string]*widget.Check
	scroll    *container.Scroll
	onChange  func()
}

func NewLanguagePanel() *LanguagePanel {
	p := &LanguagePanel{
		checks: make(map[string]*widget.Check),
	}
	return p
}

func (p *LanguagePanel) SetLanguages(languages []language.Language) {
	p.languages = languages
	p.checks = make(map[string]*widget.Check)
	p.Refresh()
}

func (p *LanguagePanel) CreateRenderer() fyne.WidgetRenderer {
	grid := container.NewGridWithColumns(2)
	for _, lang := range p.languages {
		label := lang.Name + " (" + lang.Code + ")"
		check, exists := p.checks[lang.Code]
		if !exists {
			check = widget.NewCheck(label, func(b bool) {
				if p.onChange != nil {
					p.onChange()
				}
			})
			p.checks[lang.Code] = check
		}
		grid.Add(check)
	}

	selectAllBtn := widget.NewButton("全选", func() {
		for _, c := range p.checks {
			c.SetChecked(true)
		}
	})
	deselectAllBtn := widget.NewButton("取消全选", func() {
		for _, c := range p.checks {
			c.SetChecked(false)
		}
	})

	header := container.NewHBox(
		widget.NewLabelWithStyle("🌍 目标语言", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
		selectAllBtn,
		deselectAllBtn,
	)

	scroll := container.NewVScroll(grid)
	scroll.SetMinSize(fyne.NewSize(0, 200))
	p.scroll = scroll

	box := container.NewBorder(header, nil, nil, nil, scroll)
	return widget.NewSimpleRenderer(box)
}

func (p *LanguagePanel) SelectedLanguages() []language.Language {
	var selected []language.Language
	for _, lang := range p.languages {
		if c, ok := p.checks[lang.Code]; ok && c.Checked {
			selected = append(selected, lang)
		}
	}
	return selected
}

func (p *LanguagePanel) SelectedCount() int {
	count := 0
	for _, c := range p.checks {
		if c.Checked {
			count++
		}
	}
	return count
}

func (p *LanguagePanel) SetSelectedCodes(codes []string) {
	codeSet := make(map[string]bool)
	for _, c := range codes {
		codeSet[c] = true
	}
	for code, check := range p.checks {
		check.SetChecked(codeSet[code])
	}
}

func (p *LanguagePanel) GetSelectedCodes() []string {
	var codes []string
	for code, check := range p.checks {
		if check.Checked {
			codes = append(codes, code)
		}
	}
	return codes
}

func (p *LanguagePanel) SetOnChange(fn func()) {
	p.onChange = fn
}
