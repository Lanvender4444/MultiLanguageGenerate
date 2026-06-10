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
	items map[string]*progressItem
	list  *fyne.Container
	box   *fyne.Container
}

func NewProgressPanel() *ProgressPanel {
	p := &ProgressPanel{
		items: make(map[string]*progressItem),
	}
	p.list = container.NewVBox()
	p.box = container.NewVBox(
		widget.NewLabelWithStyle("翻译进度", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		p.list,
	)
	return p
}

func (p *ProgressPanel) Container() *fyne.Container {
	return p.box
}

func (p *ProgressPanel) InitJobs(codes []string) {
	p.items = make(map[string]*progressItem)
	p.list.Objects = nil

	for _, code := range codes {
		item := &progressItem{
			code:     code,
			label:    widget.NewLabel(code),
			status:   widget.NewLabel("等待中..."),
			bar:      widget.NewProgressBar(),
			retryBtn: widget.NewButton("重试", nil),
		}
		item.retryBtn.Hide()
		item.row = container.NewHBox(
			item.label,
			item.bar,
			item.status,
			item.retryBtn,
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
		item.status.SetText(fmt.Sprintf("失败: %s", truncateError(result.Error.Error(), 50)))
		item.retryBtn.Show()
	} else {
		item.bar.SetValue(1)
		item.status.SetText("完成")
	}
	item.bar.Refresh()
	item.status.Refresh()
	item.retryBtn.Refresh()
}

func (p *ProgressPanel) SetTranslating(code string) {
	if item, ok := p.items[code]; ok {
		item.status.SetText("翻译中...")
		item.bar.SetValue(0.3)
		item.status.Refresh()
		item.bar.Refresh()
	}
}

func truncateError(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func (p *ProgressPanel) AddStatusLine(text string) {
	lbl := widget.NewLabel(text)
	p.list.Add(lbl)
	p.list.Refresh()
}