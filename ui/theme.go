package ui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// ── 双主题：棕色木板门 / 千禧年银色水汽 ──────────────────────────────────────
//
// iOS4 时代的拟物质感：真实材质背景 + 浮雕卡片 + 顶部高光釉面。
// 字体统一使用霞鹜文楷 Screen（柔和楷体笔触）。

type ThemeKind int

const (
	ThemeWood   ThemeKind = iota // 棕色木板门
	ThemeSilver                  // 银色水汽·千禧年
)

func ThemeKindFromString(s string) ThemeKind {
	if s == "silver" {
		return ThemeSilver
	}
	return ThemeWood
}

func (k ThemeKind) String() string {
	if k == ThemeSilver {
		return "silver"
	}
	return "wood"
}

func (k ThemeKind) DisplayName() string {
	if k == ThemeSilver {
		return "水汽·银"
	}
	return "木门·棕"
}

// Palette 一套主题的全部设计令牌。
type Palette struct {
	Background    color.NRGBA // 兜底背景色（纹理图之下）
	Surface       color.NRGBA // 卡片面
	SurfaceHigh   color.NRGBA // 按钮等抬升面
	InputBg       color.NRGBA // 输入框
	Border        color.NRGBA // 卡片描边
	BorderBright  color.NRGBA // 亮描边 / 滚动条
	Primary       color.NRGBA // 主操作色
	OnPrimary     color.NRGBA // 主操作色上的文字
	Success       color.NRGBA
	Error         color.NRGBA
	Warning       color.NRGBA
	TextPrimary   color.NRGBA
	TextSecondary color.NRGBA
	TextTertiary  color.NRGBA
	Selection     color.NRGBA
	Hover         color.NRGBA
	Pressed       color.NRGBA
	Shadow        color.NRGBA

	CardShadow color.NRGBA // 卡片下落影
	CardGloss  color.NRGBA // 卡片顶部釉面高光
	HeaderTint color.NRGBA // 头部纹理上的压暗/提亮罩
	DropGlow   color.NRGBA // 装饰圆（水珠 / 铜钉光晕）
	DropSpec   color.NRGBA // 装饰圆高光点
}

// 棕色木板门：胡桃木 + 牛皮纸卡片 + 黄铜主色
var woodPalette = Palette{
	Background:    rgba(0x4A, 0x32, 0x20, 0xFF),
	Surface:       rgba(0xF3, 0xE8, 0xD2, 0xF8), // 牛皮纸
	SurfaceHigh:   rgba(0xE6, 0xD5, 0xB4, 0xFF),
	InputBg:       rgba(0xFB, 0xF4, 0xE4, 0xFF),
	Border:        rgba(0x7E, 0x5F, 0x3E, 0xFF),
	BorderBright:  rgba(0xA8, 0x85, 0x5C, 0xFF),
	Primary:       rgba(0xA9, 0x6A, 0x28, 0xFF), // 黄铜
	OnPrimary:     rgba(0xFD, 0xF6, 0xE8, 0xFF),
	Success:       rgba(0x5E, 0x7E, 0x3A, 0xFF), // 橄榄绿
	Error:         rgba(0xA8, 0x40, 0x2F, 0xFF), // 砖红
	Warning:       rgba(0xC7, 0x7F, 0x2E, 0xFF),
	TextPrimary:   rgba(0x3A, 0x2B, 0x1C, 0xFF), // 深咖
	TextSecondary: rgba(0x6F, 0x57, 0x3D, 0xFF),
	TextTertiary:  rgba(0xAE, 0x97, 0x76, 0xFF),
	Selection:     rgba(0xA9, 0x6A, 0x28, 0x42),
	Hover:         rgba(0x3A, 0x2B, 0x1C, 0x12),
	Pressed:       rgba(0x3A, 0x2B, 0x1C, 0x1E),
	Shadow:        rgba(0x20, 0x12, 0x06, 0x66),

	CardShadow: rgba(0x1C, 0x0F, 0x05, 0x58),
	CardGloss:  rgba(0xFF, 0xFB, 0xF0, 0x20),
	HeaderTint: rgba(0x2A, 0x1A, 0x0C, 0x52),
	DropGlow:   rgba(0xD8, 0x9A, 0x46, 0x2A), // 铜色光晕
	DropSpec:   rgba(0xFF, 0xEE, 0xCC, 0x55),
}

// 千禧年银色水汽：拉丝银 + 半透玻璃卡片 + Aqua 蓝
var silverPalette = Palette{
	Background:    rgba(0xC7, 0xCD, 0xD6, 0xFF),
	Surface:       rgba(0xF4, 0xF7, 0xFB, 0xD8), // 半透白玻璃
	SurfaceHigh:   rgba(0xE3, 0xE9, 0xF1, 0xFF),
	InputBg:       rgba(0xFC, 0xFD, 0xFF, 0xFF),
	Border:        rgba(0xFF, 0xFF, 0xFF, 0xB4), // 亮边玻璃圈
	BorderBright:  rgba(0x8E, 0x98, 0xA8, 0xFF),
	Primary:       rgba(0x1E, 0x78, 0xD7, 0xFF), // Aqua 蓝
	OnPrimary:     rgba(0xFF, 0xFF, 0xFF, 0xFF),
	Success:       rgba(0x2F, 0xA8, 0x4F, 0xFF),
	Error:         rgba(0xD6, 0x45, 0x3A, 0xFF),
	Warning:       rgba(0xE0, 0x8A, 0x1E, 0xFF),
	TextPrimary:   rgba(0x23, 0x2A, 0x33, 0xFF), // 石板黑
	TextSecondary: rgba(0x5C, 0x66, 0x75, 0xFF),
	TextTertiary:  rgba(0x9A, 0xA4, 0xB2, 0xFF),
	Selection:     rgba(0x1E, 0x78, 0xD7, 0x46),
	Hover:         rgba(0x1E, 0x78, 0xD7, 0x14),
	Pressed:       rgba(0x1E, 0x78, 0xD7, 0x22),
	Shadow:        rgba(0x3A, 0x46, 0x58, 0x40),

	CardShadow: rgba(0x46, 0x52, 0x66, 0x36),
	CardGloss:  rgba(0xFF, 0xFF, 0xFF, 0x58),
	HeaderTint: rgba(0xFF, 0xFF, 0xFF, 0x2E),
	DropGlow:   rgba(0xCE, 0xE2, 0xF5, 0x4A), // 水珠
	DropSpec:   rgba(0xFF, 0xFF, 0xFF, 0x80),
}

var currentKind = ThemeWood

// SetThemeKind 切换全局主题（之后需重新 SetTheme 以触发刷新）。
func SetThemeKind(k ThemeKind) { currentKind = k }

// CurrentThemeKind 返回当前主题。
func CurrentThemeKind() ThemeKind { return currentKind }

// P 返回当前主题调色板。
func P() *Palette {
	if currentKind == ThemeSilver {
		return &silverPalette
	}
	return &woodPalette
}

// ── fyne.Theme 实现 ──────────────────────────────────────────────────────────

type skeuoTheme struct{}

var _ fyne.Theme = (*skeuoTheme)(nil)

// NewCustomTheme 返回当前拟物主题。
func NewCustomTheme() fyne.Theme {
	return &skeuoTheme{}
}

func (t *skeuoTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	p := P()
	switch name {
	case theme.ColorNameBackground:
		return p.Background
	case theme.ColorNameForeground:
		return p.TextPrimary
	case theme.ColorNamePrimary:
		return p.Primary
	case theme.ColorNameForegroundOnPrimary:
		return p.OnPrimary
	case theme.ColorNameButton:
		return p.SurfaceHigh
	case theme.ColorNameDisabledButton:
		return p.Surface
	case theme.ColorNameDisabled:
		return p.TextTertiary
	case theme.ColorNameInputBackground:
		return p.InputBg
	case theme.ColorNameInputBorder:
		return p.BorderBright
	case theme.ColorNamePlaceHolder:
		return p.TextTertiary
	case theme.ColorNameScrollBar:
		return p.BorderBright
	case theme.ColorNameShadow:
		return p.Shadow
	case theme.ColorNameSuccess:
		return p.Success
	case theme.ColorNameError:
		return p.Error
	case theme.ColorNameWarning:
		return p.Warning
	case theme.ColorNameSeparator:
		return p.Border
	case theme.ColorNameOverlayBackground:
		return color.NRGBA{R: p.Surface.R, G: p.Surface.G, B: p.Surface.B, A: 0xFF}
	case theme.ColorNameHeaderBackground:
		return p.SurfaceHigh
	case theme.ColorNameMenuBackground:
		return color.NRGBA{R: p.Surface.R, G: p.Surface.G, B: p.Surface.B, A: 0xFF}
	case theme.ColorNameHover:
		return p.Hover
	case theme.ColorNamePressed:
		return p.Pressed
	case theme.ColorNameFocus:
		return p.Primary
	case theme.ColorNameSelection:
		return p.Selection
	}
	return theme.DefaultTheme().Color(name, variant)
}

func (t *skeuoTheme) Font(style fyne.TextStyle) fyne.Resource {
	// 霞鹜文楷 Screen：单字重，所有样式统一使用，柔和不生硬。
	if style.Symbol {
		return theme.DefaultTheme().Font(style)
	}
	return fontWenKai
}

func (t *skeuoTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (t *skeuoTheme) Size(name fyne.ThemeSizeName) float32 {
	switch name {
	case theme.SizeNamePadding:
		return 10
	case theme.SizeNameInnerPadding:
		return 9
	case theme.SizeNameText:
		return 14 // 文楷笔画纤细，稍大一号更舒展
	case theme.SizeNameHeadingText:
		return 22
	case theme.SizeNameSubHeadingText:
		return 17
	case theme.SizeNameCaptionText:
		return 12
	case theme.SizeNameInputRadius:
		return 10
	case theme.SizeNameSelectionRadius:
		return 8
	case theme.SizeNameScrollBarSmall:
		return 4
	case theme.SizeNameScrollBar:
		return 6
	case theme.SizeNameSeparatorThickness:
		return 1
	}
	return theme.DefaultTheme().Size(name)
}
