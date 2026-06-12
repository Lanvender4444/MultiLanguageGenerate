package ui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// 霞鹜文楷 Screen（LXGW WenKai Screen），SIL OFL 1.1 开源协议。
// 楷体笔触柔和温润，含完整 CJK 字形。
// 获取：https://github.com/lxgw/LxgwWenkai-Screen/releases
//
//go:embed assets/LXGWWenKaiScreen.ttf
var lxgwWenKaiTTF []byte

var fontWenKai = fyne.NewStaticResource("LXGWWenKaiScreen.ttf", lxgwWenKaiTTF)
