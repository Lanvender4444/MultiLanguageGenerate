package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"github.com/yourname/MultiLanguageGenerate/internal/translator"
)

type progressItem struct {
	code     string
	label    *widget.Label
	progress *widget.ProgressBar
	status   *widget.Label
	retryBtn *widget.Button
}

type ProgressPanel struct {
	widget.BaseWidget
	items map[string]*progressItem
	box   *fyne.Container
}

func NewProgressPanel() *ProgressPanel {
	p := &ProgressPanel{
		items: make(map[string]*progressItem),
		box:   container.NewVBox(),
	}
	return p
}

func (p *ProgressPanel) CreateRenderer() fyne.WidgetRenderer {
	header := widget.NewLabelWithStyle("📊 进度", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	wrapper := container.NewBorder(header, nil, nil, nil, container.NewVScroll(p.box))
	return widget.NewSimpleRenderer(wrapper)
}

func (p *ProgressPanel) InitJobs(codes []string) {
	p.items = make(map[string]*progressItem)
	p.box.Objects = nil

	for _, code := range codes {
		item := &progressItem{
			code:     code,
			label:    widget.NewLabel(code),
			progress: widget.NewProgressBar(),
			status:   widget.NewLabel("⏳ 等待中..."),
			retryBtn: widget.NewButton("重试", nil),
		}
		item.retryBtn.Hide()
		p.items[code] = item
		p.box.Add(container.NewHBox(
			item.label,
			item.progress,
			item.status,
			item.retryBtn,
		))
	}
	p.box.Refresh()
}

func (p *ProgressPanel) UpdateResult(result translator.Result) {
	item, ok := p.items[result.TargetCode]
	if !ok {
		return
	}

	if result.Error != nil {
		item.progress.SetValue(1)
		item.status.SetText(fmt.Sprintf("❌ %s", result.Error.Error()))
		item.retryBtn.Show()
	} else {
		item.progress.SetValue(1)
		item.status.SetText(fmt.Sprintf("✅ %s", result.OutputPath))
	}
	item.progress.Refresh()
	item.status.Refresh()
	item.retryBtn.Refresh()
}

func (p *ProgressPanel) SetTranslating(code string) {
	if item, ok := p.items[code]; ok {
		item.status.SetText("⏳ 翻译中...")
		item.progress.SetValue(0.5)
		item.status.Refresh()
		item.progress.Refresh()
	}
}

func (p *ProgressPanel) SetRetryFunc(code string, fn func()) {
	if item, ok := p.items[code]; ok {
		item.retryBtn.OnTapped = fn
	}
}

func (p *ProgressPanel) AllDone() bool {
	for _, item := range p.items {
		t := item.status.Text
		if len(t) >= 2 && t[:2] == "⏳" {
			return false
		}
	}
	return true
}
