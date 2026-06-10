package ui

import (
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

type SourcePanel struct {
	widget.BaseWidget
	entry      *widget.Entry
	selectBtn  *widget.Button
	langSelect *widget.Select
	window     fyne.Window

	SourceFile     string
	SourceLanguage string
}

func NewSourcePanel(window fyne.Window) *SourcePanel {
	p := &SourcePanel{
		window:         window,
		SourceLanguage: "auto",
	}

	p.entry = widget.NewEntry()
	p.entry.SetPlaceHolder("选择或输入源文件路径...")
	p.entry.OnChanged = func(s string) {
		p.SourceFile = s
	}

	p.selectBtn = widget.NewButton("选择文件", func() {
		dialog.ShowFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			path := reader.URI().Path()
			p.entry.SetText(path)
			p.SourceFile = path
			reader.Close()
		}, window)
	})

	languages := []string{"AI 自动检测"}
	knownLangs := []string{
		"中文（简体）", "中文（繁體）", "English", "日本語", "한국어",
		"español", "français", "Deutsch", "português", "русский",
		"العربية", "हिन्दी", "Bahasa Indonesia", "italiano", "Nederlands",
	}
	languages = append(languages, knownLangs...)

	p.langSelect = widget.NewSelect(languages, func(s string) {
		if s == "AI 自动检测" {
			p.SourceLanguage = "auto"
		} else {
			p.SourceLanguage = s
		}
	})
	p.langSelect.SetSelected("AI 自动检测")

	return p
}

func (p *SourcePanel) CreateRenderer() fyne.WidgetRenderer {
	box := container.NewVBox(
		widget.NewLabelWithStyle("📄 源文件", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		container.NewBorder(nil, nil, nil, p.selectBtn, p.entry),
		container.NewHBox(
			widget.NewLabel("源语言:"),
			p.langSelect,
		),
	)
	return widget.NewSimpleRenderer(container.NewVScroll(box))
}

func (p *SourcePanel) ReadSourceContent() (string, error) {
	if p.SourceFile == "" {
		return "", nil
	}
	data, err := os.ReadFile(p.SourceFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (p *SourcePanel) SourceDir() string {
	if p.SourceFile == "" {
		return ""
	}
	return filepath.Dir(p.SourceFile)
}
