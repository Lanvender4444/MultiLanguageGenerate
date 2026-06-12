package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

// ── GlassPanel ───────────────────────────────────────────────────────────────
//
// 拟物卡片：下落影 + 描边 + 材质面 + 顶部釉面高光。
// 木门主题呈牛皮纸便签，水汽主题呈半透 Aqua 玻璃。

type GlassPanel struct {
	widget.BaseWidget
	Title   string
	content fyne.CanvasObject
}

func NewGlassPanel(title string, content fyne.CanvasObject) *GlassPanel {
	g := &GlassPanel{Title: title, content: content}
	g.ExtendBaseWidget(g)
	return g
}

func (g *GlassPanel) CreateRenderer() fyne.WidgetRenderer {
	p := P()

	shadow := canvas.NewRectangle(p.CardShadow)
	shadow.CornerRadius = 16

	border := canvas.NewRectangle(p.Border)
	border.CornerRadius = 15

	bg := canvas.NewRectangle(p.Surface)
	bg.CornerRadius = 14

	gloss := canvas.NewRectangle(p.CardGloss)
	gloss.CornerRadius = 12

	var titleTxt *canvas.Text
	if g.Title != "" {
		titleTxt = canvas.NewText(g.Title, p.TextSecondary)
		titleTxt.TextSize = 12
		titleTxt.TextStyle = fyne.TextStyle{Bold: true}
	}

	objs := []fyne.CanvasObject{shadow, border, bg, gloss}
	if titleTxt != nil {
		objs = append(objs, titleTxt)
	}
	objs = append(objs, g.content)

	return &glassPanelRenderer{
		shadow:  shadow,
		border:  border,
		bg:      bg,
		gloss:   gloss,
		title:   titleTxt,
		content: g.content,
		all:     objs,
		panel:   g,
	}
}

const (
	gpPad    = float32(16)
	gpTitleH = float32(18)
)

type glassPanelRenderer struct {
	shadow  *canvas.Rectangle
	border  *canvas.Rectangle
	bg      *canvas.Rectangle
	gloss   *canvas.Rectangle
	title   *canvas.Text
	content fyne.CanvasObject
	all     []fyne.CanvasObject
	panel   *GlassPanel
}

func (r *glassPanelRenderer) Layout(size fyne.Size) {
	// 下落影：向下偏移 3px
	r.shadow.Move(fyne.NewPos(0, 3))
	r.shadow.Resize(size)

	r.border.Move(fyne.NewPos(0, 0))
	r.border.Resize(size)

	r.bg.Move(fyne.NewPos(1.5, 1.5))
	r.bg.Resize(fyne.NewSize(size.Width-3, size.Height-3))

	// 顶部釉面：覆盖上方约 36%
	r.gloss.Move(fyne.NewPos(3, 3))
	r.gloss.Resize(fyne.NewSize(size.Width-6, size.Height*0.36))

	contentY := gpPad
	if r.title != nil {
		r.title.Move(fyne.NewPos(gpPad, gpPad-2))
		r.title.Resize(fyne.NewSize(size.Width-gpPad*2, gpTitleH))
		contentY = gpPad + gpTitleH + 4
	}

	r.content.Move(fyne.NewPos(gpPad, contentY))
	r.content.Resize(fyne.NewSize(size.Width-gpPad*2, size.Height-contentY-gpPad))
}

func (r *glassPanelRenderer) MinSize() fyne.Size {
	extra := gpPad * 2
	if r.title != nil {
		extra += gpTitleH + 4
	}
	cm := r.content.MinSize()
	return fyne.NewSize(cm.Width+gpPad*2, cm.Height+extra)
}

func (r *glassPanelRenderer) Refresh() {
	p := P()
	r.shadow.FillColor = p.CardShadow
	r.border.FillColor = p.Border
	r.bg.FillColor = p.Surface
	r.gloss.FillColor = p.CardGloss
	if r.title != nil {
		r.title.Color = p.TextSecondary
	}
	canvas.Refresh(r.panel)
}

func (r *glassPanelRenderer) Destroy()                     {}
func (r *glassPanelRenderer) Objects() []fyne.CanvasObject { return r.all }

// ── WaterDropBg ──────────────────────────────────────────────────────────────
//
// 全幅材质背景：程序化纹理图。水汽主题上再叠半透水珠（带高光点），
// 木门主题纹理本身已含板缝与铜钉，不叠加水珠。

type dropSpec struct {
	xFrac, yFrac float32
	radius       float32
}

var waterDropSpecs = []dropSpec{
	{0.07, 0.10, 88},
	{0.86, 0.08, 58},
	{0.93, 0.74, 72},
	{0.14, 0.83, 46},
	{0.60, 0.92, 62},
	{0.47, 0.36, 34},
	{0.32, 0.52, 22},
}

type WaterDropBg struct {
	widget.BaseWidget
}

func NewWaterDropBg() *WaterDropBg {
	w := &WaterDropBg{}
	w.ExtendBaseWidget(w)
	return w
}

func (w *WaterDropBg) CreateRenderer() fyne.WidgetRenderer {
	tex := canvas.NewImageFromImage(BackgroundTexture(CurrentThemeKind()))
	tex.FillMode = canvas.ImageFillStretch
	tex.ScaleMode = canvas.ImageScaleSmooth

	n := len(waterDropSpecs)
	drops := make([]*canvas.Circle, n)
	spec := make([]*canvas.Circle, n)
	all := []fyne.CanvasObject{tex}

	for i := range waterDropSpecs {
		drops[i] = canvas.NewCircle(color.Transparent)
		spec[i] = canvas.NewCircle(color.Transparent)
		all = append(all, drops[i], spec[i])
	}

	r := &wdbRenderer{tex: tex, drops: drops, specular: spec, all: all, w: w}
	r.applyTheme()
	return r
}

type wdbRenderer struct {
	tex      *canvas.Image
	drops    []*canvas.Circle
	specular []*canvas.Circle
	all      []fyne.CanvasObject
	w        *WaterDropBg
}

func (r *wdbRenderer) applyTheme() {
	r.tex.Image = BackgroundTexture(CurrentThemeKind())

	p := P()
	dropC, specC := color.Color(color.Transparent), color.Color(color.Transparent)
	if CurrentThemeKind() == ThemeSilver {
		dropC, specC = p.DropGlow, p.DropSpec
	}
	for i := range r.drops {
		r.drops[i].FillColor = dropC
		r.specular[i].FillColor = specC
	}
}

func (r *wdbRenderer) Layout(size fyne.Size) {
	r.tex.Resize(size)
	for i, sp := range waterDropSpecs {
		cx := sp.xFrac * size.Width
		cy := sp.yFrac * size.Height
		rad := sp.radius

		r.drops[i].Move(fyne.NewPos(cx-rad, cy-rad))
		r.drops[i].Resize(fyne.NewSize(rad*2, rad*2))

		hlR := rad * 0.27
		r.specular[i].Move(fyne.NewPos(cx+rad*0.30, cy-rad*0.52))
		r.specular[i].Resize(fyne.NewSize(hlR*2, hlR*2))
	}
}

func (r *wdbRenderer) MinSize() fyne.Size { return fyne.NewSize(0, 0) }

func (r *wdbRenderer) Refresh() {
	r.applyTheme()
	r.tex.Refresh()
	canvas.Refresh(r.w)
}

func (r *wdbRenderer) Destroy()              {}
func (r *wdbRenderer) Objects() []fyne.CanvasObject { return r.all }

// ── AppHeader ────────────────────────────────────────────────────────────────
//
// 拟物顶栏：材质底色 + 顶部釉面 + 标题 + 主题切换钮。
// 木门主题为深胡桃门楣配奶油字，水汽主题为亮银釉面配石板字。

type AppHeader struct {
	widget.BaseWidget
	OnToggleTheme func()

	toggleBtn *widget.Button
}

func NewAppHeader() *AppHeader {
	h := &AppHeader{}
	h.toggleBtn = widget.NewButton(CurrentThemeKind().DisplayName(), func() {
		if h.OnToggleTheme != nil {
			h.OnToggleTheme()
		}
	})
	h.ExtendBaseWidget(h)
	return h
}

// headerTextColors 返回 (标题色, 副标题色)：木门顶栏深色需配亮字。
func headerTextColors() (color.NRGBA, color.NRGBA) {
	if CurrentThemeKind() == ThemeWood {
		return rgba(0xF6, 0xEC, 0xD8, 0xFF), rgba(0xCD, 0xB6, 0x92, 0xFF)
	}
	p := P()
	return p.TextPrimary, p.TextSecondary
}

func (h *AppHeader) CreateRenderer() fyne.WidgetRenderer {
	p := P()

	base := canvas.NewRectangle(p.Background)
	tint := canvas.NewRectangle(p.HeaderTint)
	gloss := canvas.NewRectangle(p.CardGloss)

	tc, sc := headerTextColors()
	title := canvas.NewText("MultiLanguage", tc)
	title.TextSize = 23
	title.TextStyle = fyne.TextStyle{Bold: true}

	sub := canvas.NewText("AI 多语言翻译工具", sc)
	sub.TextSize = 13

	sep := canvas.NewRectangle(p.Border)

	all := []fyne.CanvasObject{base, tint, gloss, title, sub, sep, h.toggleBtn}

	return &appHeaderRenderer{
		base: base, tint: tint, gloss: gloss,
		title: title, sub: sub, sep: sep,
		btn: h.toggleBtn,
		all: all, h: h,
	}
}

type appHeaderRenderer struct {
	base  *canvas.Rectangle
	tint  *canvas.Rectangle
	gloss *canvas.Rectangle
	title *canvas.Text
	sub   *canvas.Text
	sep   *canvas.Rectangle
	btn   *widget.Button
	all   []fyne.CanvasObject
	h     *AppHeader
}

func (r *appHeaderRenderer) Layout(size fyne.Size) {
	r.base.Resize(size)
	r.tint.Resize(size)

	// 顶部釉面高光带：上半段
	r.gloss.Resize(fyne.NewSize(size.Width, size.Height*0.45))

	r.title.Move(fyne.NewPos(20, 12))
	r.title.Resize(fyne.NewSize(size.Width-220, 30))

	r.sub.Move(fyne.NewPos(20, 44))
	r.sub.Resize(fyne.NewSize(size.Width-220, 18))

	btnMin := r.btn.MinSize()
	r.btn.Move(fyne.NewPos(size.Width-btnMin.Width-16, (size.Height-btnMin.Height)/2))
	r.btn.Resize(btnMin)

	r.sep.Move(fyne.NewPos(0, size.Height-1))
	r.sep.Resize(fyne.NewSize(size.Width, 1))
}

func (r *appHeaderRenderer) MinSize() fyne.Size {
	return fyne.NewSize(0, 74)
}

func (r *appHeaderRenderer) Refresh() {
	p := P()
	r.base.FillColor = p.Background
	r.tint.FillColor = p.HeaderTint
	r.gloss.FillColor = p.CardGloss
	tc, sc := headerTextColors()
	r.title.Color = tc
	r.sub.Color = sc
	r.sep.FillColor = p.Border
	r.btn.SetText(CurrentThemeKind().DisplayName())
	canvas.Refresh(r.h)
}

func (r *appHeaderRenderer) Destroy()              {}
func (r *appHeaderRenderer) Objects() []fyne.CanvasObject { return r.all }

// ── FixedSpacer ──────────────────────────────────────────────────────────────
//
// 透明定高占位，用于卡片之间的垂直留白（不遮挡材质背景）。

type FixedSpacer struct {
	widget.BaseWidget
	h float32
}

func NewFixedSpacer(h float32) *FixedSpacer {
	s := &FixedSpacer{h: h}
	s.ExtendBaseWidget(s)
	return s
}

func (s *FixedSpacer) MinSize() fyne.Size {
	return fyne.NewSize(0, s.h)
}

func (s *FixedSpacer) CreateRenderer() fyne.WidgetRenderer {
	rect := canvas.NewRectangle(color.Transparent)
	return &fixedSpacerRenderer{rect: rect, spacer: s}
}

type fixedSpacerRenderer struct {
	rect   *canvas.Rectangle
	spacer *FixedSpacer
}

func (r *fixedSpacerRenderer) Layout(size fyne.Size)        { r.rect.Resize(size) }
func (r *fixedSpacerRenderer) MinSize() fyne.Size           { return fyne.NewSize(0, r.spacer.h) }
func (r *fixedSpacerRenderer) Refresh()                     { canvas.Refresh(r.spacer) }
func (r *fixedSpacerRenderer) Destroy()                     {}
func (r *fixedSpacerRenderer) Objects() []fyne.CanvasObject { return []fyne.CanvasObject{r.rect} }
