package ui

import (
	"image"
	"image/color"
	"math"
)

// ── 程序化纹理生成 ────────────────────────────────────────────────────────────
//
// 不依赖任何外部图片资源，启动时绘制一次并缓存：
//   - woodTexture   棕色木板门：垂直拼板、年轮纹、板缝阴影、铜钉、暗角
//   - silverTexture 千禧年银色水汽：拉丝金属渐变、水雾光斑、冷蓝高光

const (
	texW = 768
	texH = 1024
)

var (
	woodTexCache   *image.RGBA
	silverTexCache *image.RGBA
)

// BackgroundTexture 返回当前主题的背景纹理（惰性生成并缓存）。
func BackgroundTexture(kind ThemeKind) image.Image {
	if kind == ThemeWood {
		if woodTexCache == nil {
			woodTexCache = woodTexture(texW, texH)
		}
		return woodTexCache
	}
	if silverTexCache == nil {
		silverTexCache = silverTexture(texW, texH)
	}
	return silverTexCache
}

// ── 噪声工具 ─────────────────────────────────────────────────────────────────

// hash01 把整点坐标映射到 [0,1) 的确定性伪随机值。
func hash01(x, y, seed int) float64 {
	h := uint32(x)*374761393 + uint32(y)*668265263 + uint32(seed)*1442695041
	h = (h ^ (h >> 13)) * 1274126177
	h ^= h >> 16
	return float64(h) / float64(math.MaxUint32)
}

func smoothstep(t float64) float64 { return t * t * (3 - 2*t) }

// vnoise 双线性插值的值噪声。
func vnoise(x, y float64, seed int) float64 {
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	tx, ty := smoothstep(x-float64(x0)), smoothstep(y-float64(y0))

	a := hash01(x0, y0, seed)
	b := hash01(x0+1, y0, seed)
	c := hash01(x0, y0+1, seed)
	d := hash01(x0+1, y0+1, seed)

	top := a + (b-a)*tx
	bot := c + (d-c)*tx
	return top + (bot-top)*ty
}

// fbm 分形布朗运动：叠加多个倍频的值噪声。
func fbm(x, y float64, octaves, seed int) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * vnoise(x, y, seed+i*101)
		norm += amp
		x *= 2.03
		y *= 2.03
		amp *= 0.5
	}
	return sum / norm
}

func clamp255(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

// ── 棕色木板门 ────────────────────────────────────────────────────────────────

func woodTexture(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	const plankW = 96.0
	// 基准胡桃木色
	baseR, baseG, baseB := 96.0, 62.0, 36.0

	for y := 0; y < h; y++ {
		fy := float64(y)
		for x := 0; x < w; x++ {
			fx := float64(x)

			plank := int(fx / plankW)
			inPlank := math.Mod(fx, plankW)

			// 每块板独立的明度与色相偏移
			pv := hash01(plank, 7, 42)
			lightness := lerp(0.88, 1.10, pv)
			warm := lerp(-6, 8, hash01(plank, 13, 99))

			// 年轮：沿板内 x 的正弦 + 噪声扰动，竖直方向拉长
			turb := fbm(fx*0.012, fy*0.0028, 4, 1000+plank*17)
			grain := math.Sin((inPlank*0.55 + turb*38.0))
			grain = grain * grain // 锐化成窄条纹
			grainShade := lerp(-14, 9, grain)

			// 细密直纹
			streak := fbm(fx*0.30, fy*0.012, 3, 2000+plank*31)
			streakShade := lerp(-8, 8, streak)

			// 长波浪明暗（木色不均）
			cloud := fbm(fx*0.004, fy*0.002, 3, 3000)
			cloudShade := lerp(-12, 12, cloud)

			r := (baseR + warm + grainShade + streakShade + cloudShade) * lightness
			g := (baseG + warm*0.6 + grainShade*0.9 + streakShade + cloudShade) * lightness
			b := (baseB + grainShade*0.7 + streakShade*0.8 + cloudShade) * lightness

			// 板缝：左右 3px 内压暗，缝左侧再补一条受光边
			edge := math.Min(inPlank, plankW-inPlank)
			if edge < 3.0 {
				k := edge / 3.0
				dark := lerp(0.42, 1.0, k)
				r *= dark
				g *= dark
				b *= dark
			} else if inPlank >= 3.0 && inPlank < 5.0 {
				r += 10
				g += 7
				b += 4
			}

			// 暗角（vignette），让中心更聚焦
			dx := (fx/float64(w) - 0.5) * 2
			dy := (fy/float64(h) - 0.5) * 2
			vig := 1.0 - 0.22*(dx*dx+dy*dy)
			r *= vig
			g *= vig
			b *= vig

			i := img.PixOffset(x, y)
			img.Pix[i+0] = clamp255(r)
			img.Pix[i+1] = clamp255(g)
			img.Pix[i+2] = clamp255(b)
			img.Pix[i+3] = 0xFF
		}
	}

	// 铜钉：每块板上下各一颗
	nPlanks := int(float64(w) / plankW)
	for p := 0; p < nPlanks; p++ {
		cx := int(float64(p)*plankW + plankW/2)
		drawRivet(img, cx, 36)
		drawRivet(img, cx, h-36)
	}

	return img
}

// drawRivet 画一颗带高光的铜钉。
func drawRivet(img *image.RGBA, cx, cy int) {
	const rad = 7.0
	b := img.Bounds()
	for y := cy - int(rad) - 1; y <= cy+int(rad)+1; y++ {
		for x := cx - int(rad) - 1; x <= cx+int(rad)+1; x++ {
			if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
				continue
			}
			dx, dy := float64(x-cx), float64(y-cy)
			d := math.Sqrt(dx*dx + dy*dy)
			if d > rad {
				continue
			}
			// 球面光照：左上受光
			nl := (-dx - dy) / (rad * 1.4142)
			shade := 0.55 + 0.45*nl
			r := 168 * shade
			g := 116 * shade
			bb := 58 * shade
			// 边缘暗一圈
			if d > rad-1.5 {
				r *= 0.5
				g *= 0.5
				bb *= 0.5
			}
			i := img.PixOffset(x, y)
			img.Pix[i+0] = clamp255(r)
			img.Pix[i+1] = clamp255(g)
			img.Pix[i+2] = clamp255(bb)
			img.Pix[i+3] = 0xFF
		}
	}
}

// ── 千禧年银色水汽 ────────────────────────────────────────────────────────────

func silverTexture(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < h; y++ {
		fy := float64(y) / float64(h)
		// 三段式银色纵向渐变：顶部亮 → 中部银 → 底部冷灰
		var r, g, b float64
		if fy < 0.42 {
			t := fy / 0.42
			r = lerp(244, 203, t)
			g = lerp(247, 209, t)
			b = lerp(250, 218, t)
		} else {
			t := (fy - 0.42) / 0.58
			r = lerp(203, 172, t)
			g = lerp(209, 180, t)
			b = lerp(218, 194, t)
		}

		for x := 0; x < w; x++ {
			fx := float64(x)

			// 水平拉丝：按行抖动 + 细噪声
			row := lerp(-5, 5, hash01(0, y, 77))
			fine := lerp(-4, 4, vnoise(fx*0.9, float64(y)*0.9, 555))
			brush := lerp(-3, 3, fbm(fx*0.002, float64(y)*0.45, 2, 888))

			rr := r + row + fine + brush
			gg := g + row + fine + brush
			bb := b + row + fine + brush + 2 // 微微偏冷

			i := img.PixOffset(x, y)
			img.Pix[i+0] = clamp255(rr)
			img.Pix[i+1] = clamp255(gg)
			img.Pix[i+2] = clamp255(bb)
			img.Pix[i+3] = 0xFF
		}
	}

	// 水汽光斑：大而柔的白雾 + 冷蓝光晕
	type blob struct {
		fx, fy, rad float64
		cr, cg, cb  float64
		amp         float64
	}
	blobs := []blob{
		{0.18, 0.16, 300, 255, 255, 255, 0.16},
		{0.85, 0.10, 220, 240, 250, 255, 0.14},
		{0.78, 0.78, 280, 255, 255, 255, 0.12},
		{0.10, 0.86, 200, 235, 245, 255, 0.10},
		{0.50, 0.45, 360, 225, 240, 255, 0.08}, // 中央冷蓝水汽
	}
	for _, bl := range blobs {
		cx, cy := bl.fx*float64(w), bl.fy*float64(h)
		r0 := bl.rad
		x0, x1 := int(cx-r0), int(cx+r0)
		y0, y1 := int(cy-r0), int(cy+r0)
		for y := y0; y <= y1; y++ {
			if y < 0 || y >= h {
				continue
			}
			for x := x0; x <= x1; x++ {
				if x < 0 || x >= w {
					continue
				}
				dx, dy := float64(x)-cx, float64(y)-cy
				d := math.Sqrt(dx*dx+dy*dy) / r0
				if d >= 1 {
					continue
				}
				a := bl.amp * (1 - d) * (1 - d)
				i := img.PixOffset(x, y)
				img.Pix[i+0] = clamp255(float64(img.Pix[i+0])*(1-a) + bl.cr*a)
				img.Pix[i+1] = clamp255(float64(img.Pix[i+1])*(1-a) + bl.cg*a)
				img.Pix[i+2] = clamp255(float64(img.Pix[i+2])*(1-a) + bl.cb*a)
			}
		}
	}

	return img
}

// rgba 便捷构造。
func rgba(r, g, b, a uint8) color.NRGBA { return color.NRGBA{R: r, G: g, B: b, A: a} }
