package knowledge

import (
	"strings"
	"unicode"
)

// tokenize 把文本切成检索用 token。
//
// 针对中英混排做了处理：
//   - 拉丁字母/数字：按连续片段切词并小写化；
//   - CJK 字符：同时产出单字(unigram)和相邻双字(bigram)，
//     兼顾召回与精度（中文无空格分词，bigram 能近似词语匹配）。
func tokenize(s string) []string {
	s = strings.ToLower(s)
	runes := []rune(s)
	var toks []string
	var cur []rune

	flush := func() {
		if len(cur) > 0 {
			toks = append(toks, string(cur))
			cur = cur[:0]
		}
	}

	for i, r := range runes {
		switch {
		case isCJK(r):
			flush()
			toks = append(toks, string(r)) // unigram
			if i+1 < len(runes) && isCJK(runes[i+1]) {
				toks = append(toks, string([]rune{r, runes[i+1]})) // bigram
			}
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			cur = append(cur, r)
		default:
			flush()
		}
	}
	flush()
	return toks
}

// isCJK 判断是否中日韩表意/假名字符
func isCJK(r rune) bool {
	return unicode.Is(unicode.Han, r) ||
		unicode.Is(unicode.Hiragana, r) ||
		unicode.Is(unicode.Katakana, r) ||
		unicode.Is(unicode.Hangul, r)
}
