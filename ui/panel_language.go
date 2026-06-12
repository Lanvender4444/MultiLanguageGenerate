package ui

import (
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/language"
)

type LanguagePanel struct {
	checks      []*widget.Check
	codes       []string
	names       []string
	searchEntry *widget.Entry
	box         *fyne.Container
	onChange    func()
}

func NewLanguagePanel() *LanguagePanel {
	return &LanguagePanel{
		onChange:    func() {},
		searchEntry: widget.NewEntry(),
	}
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

	p.searchEntry.SetPlaceHolder("搜索语言（名称或代码）...")
	p.searchEntry.OnChanged = func(s string) {
		p.rebuildGrid()
	}

	selectAllBtn := widget.NewButton("全选", func() {
		for _, c := range p.checks {
			c.SetChecked(true)
		}
		p.box.Refresh()
	})
	deselectAllBtn := widget.NewButton("取消全选", func() {
		for _, c := range p.checks {
			c.SetChecked(false)
		}
		p.box.Refresh()
	})

	header := container.NewHBox(
		layout.NewSpacer(),
		selectAllBtn,
		deselectAllBtn,
	)

	p.box.Add(p.searchEntry)
	p.box.Add(header)
	p.rebuildGrid()
	p.box.Refresh()
}

func (p *LanguagePanel) rebuildGrid() {
	filter := strings.ToLower(strings.TrimSpace(p.searchEntry.Text))

	col1 := container.NewVBox()
	col2 := container.NewVBox()
	idx := 0
	for i, chk := range p.checks {
		if filter == "" || strings.Contains(strings.ToLower(p.names[i]), filter) || strings.Contains(strings.ToLower(p.codes[i]), filter) {
			if idx%2 == 0 {
				col1.Add(chk)
			} else {
				col2.Add(chk)
			}
			idx++
		}
	}

	grid := container.NewGridWithColumns(2, col1, col2)
	scroll := container.NewVScroll(grid)
	scroll.SetMinSize(fyne.NewSize(0, 250))

	if len(p.box.Objects) >= 3 {
		p.box.Objects[2] = scroll
	} else {
		p.box.Add(scroll)
	}
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
