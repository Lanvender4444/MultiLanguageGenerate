package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/language"
)

type LanguagePanel struct {
	checks   []*widget.Check
	codes    []string
	names    []string
	box      *fyne.Container
	onChange func()
}

func NewLanguagePanel() *LanguagePanel {
	return &LanguagePanel{onChange: func() {}}
}

func (p *LanguagePanel) SetLanguages(languages []language.Language) {
	p.checks = nil
	p.codes = nil
	p.names = nil

	for _, lang := range languages {
		code := lang.Code
		name := lang.Name
		p.codes = append(p.codes, code)
		p.names = append(p.names, name)
		chk := widget.NewCheck(name+" ("+code+")", func(_ bool) {
			if p.onChange != nil {
				p.onChange()
			}
		})
		p.checks = append(p.checks, chk)
	}

	p.rebuild()
}

func (p *LanguagePanel) rebuild() {
	if p.box == nil {
		p.box = container.NewVBox()
	}
	p.box.Objects = nil

	col1 := container.NewVBox()
	col2 := container.NewVBox()
	for i, chk := range p.checks {
		if i%2 == 0 {
			col1.Add(chk)
		} else {
			col2.Add(chk)
		}
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
		layout.NewSpacer(),
		selectAllBtn,
		deselectAllBtn,
	)

	grid := container.NewGridWithColumns(2, col1, col2)
	scroll := container.NewVScroll(grid)
	scroll.SetMinSize(fyne.NewSize(0, 250))

	p.box.Add(header)
	p.box.Add(scroll)
	p.box.Refresh()
}

func (p *LanguagePanel) Container() *fyne.Container {
	if p.box == nil {
		p.box = container.NewVBox(widget.NewLabel("加载语言列表中…"))
	}
	return p.box
}

func (p *LanguagePanel) SelectedLanguages() []language.Language {
	var selected []language.Language
	for i, chk := range p.checks {
		if chk.Checked {
			selected = append(selected, language.Language{Code: p.codes[i], Name: p.names[i]})
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
	for i, chk := range p.checks {
		chk.SetChecked(codeSet[p.codes[i]])
	}
}

func (p *LanguagePanel) GetSelectedCodes() []string {
	var codes []string
	for i, chk := range p.checks {
		if chk.Checked {
			codes = append(codes, p.codes[i])
		}
	}
	return codes
}

func (p *LanguagePanel) SetOnChange(fn func()) {
	p.onChange = fn
}
