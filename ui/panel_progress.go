package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/Lanvender4444/MultiLanguageGenerate/internal/translator"
)

type progressItem struct {
	code     string
	label    *widget.Label
	status   *widget.Label
	bar      *widget.ProgressBar
	retryBtn *widget.Button
	row      *fyne.Container
}

type ProgressPanel struct {
	items    map[string]*progressItem
	list     *fyne.Container
	logBox   *fyne.Container
	box      *fyne.Container
}

func NewProgressPanel() *ProgressPanel {
	p := &ProgressPanel{items: make(map[string]*progressItem)}
	p.list = container.NewVBox()
	p.logBox = container.NewVBox()

	p.box = container.NewVBox(p.list, p.logBox)
	return p
}

func (p *ProgressPanel) Container() *fyne.Container {
	return p.box
}

func (p *ProgressPanel) InitJobs(codes []string) {
	p.items = make(map[string]*progressItem)
	p.list.Objects = nil
	p.logBox.Objects = nil

	for _, code := range codes {
		item := &progressItem{
			code:     code,
			label:    widget.NewLabel(code),
			status:   widget.NewLabel("等待中…"),
			bar:      widget.NewProgressBar(),
			retryBtn: widget.NewButton("重试", nil),
		}
		item.retryBtn.Hide()

		// Border layout: label left, status right, bar fills centre
		item.row = container.NewBorder(
			nil, nil,
			item.label,
			item.retryBtn,
			container.NewBorder(nil, nil, nil, item.status, item.bar),
		)

		p.items[code] = item
		p.list.Add(item.row)
	}
	p.list.Refresh()
}

func (p *ProgressPanel) UpdateResult(result translator.Result) {
	item, ok := p.items[result.TargetCode]
	if !ok {
		return
	}

	if result.Error != nil {
		item.bar.SetValue(1)
		item.status.SetText(fmt.Sprintf("失败: %s", truncateErr(result.Error.Error(), 40)))
		item.retryBtn.Show()
	} else {
		item.bar.SetValue(1)
		item.status.SetText("✓ 完成")
	}
	item.bar.Refresh()
	item.status.Refresh()
	item.retryBtn.Refresh()
}

func (p *ProgressPanel) SetTranslating(code string) {
	if item, ok := p.items[code]; ok {
		item.status.SetText("翻译中…")
		item.bar.SetValue(0.3)
		item.status.Refresh()
		item.bar.Refresh()
	}
}

func (p *ProgressPanel) AddStatusLine(text string) {
	p.logBox.Add(widget.NewLabel(text))
	p.logBox.Refresh()
}

func truncateErr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
