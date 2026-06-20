// Package glossary 实现"专业名词解析"。
//
// 它维护一份 JSON 词表：对特定专业名词、人物、品牌等，按不同目标语言给出
// 指定译法与适用上下文。翻译时这些约束会被注入到 LLM 的系统提示中，保证
// 术语翻译的一致性与专业性。词表既可手工维护，也可由 AI 依据知识库生成
// （见 generate.go）。
package glossary

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Entry 一个专业名词条目
type Entry struct {
	Term        string            `json:"term"`                  // 源语言中的名词/人物/术语
	Type        string            `json:"type,omitempty"`        // 分类：person/place/tech/brand/term...
	Context     string            `json:"context,omitempty"`     // 适用上下文/说明（消歧用）
	Aliases     []string          `json:"aliases,omitempty"`     // 同义/别名
	Translation map[string]string `json:"translation"`           // 目标语言代码 -> 指定译法
}

// Glossary 一份词表
type Glossary struct {
	Name        string  `json:"name,omitempty"`
	Description string  `json:"description,omitempty"`
	Entries     []Entry `json:"entries"`
}

// Load 从 JSON 文件读取词表
func Load(path string) (*Glossary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var g Glossary
	if err := json.Unmarshal(data, &g); err != nil {
		return nil, fmt.Errorf("parse glossary json: %w", err)
	}
	return &g, nil
}

// Save 把词表写入 JSON 文件（带缩进，便于人工编辑）
func (g *Glossary) Save(path string) error {
	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// RenderForTarget 为某目标语言生成可注入翻译提示的术语约束块。
// 只包含对该语言有指定译法的条目；无相关条目时返回空串。
func (g *Glossary) RenderForTarget(targetCode string) string {
	if g == nil || len(g.Entries) == 0 {
		return ""
	}
	var lines []string
	for _, e := range g.Entries {
		t, ok := e.Translation[targetCode]
		if !ok || strings.TrimSpace(t) == "" {
			continue
		}
		line := fmt.Sprintf(`- "%s" => "%s"`, e.Term, t)
		var notes []string
		if e.Type != "" {
			notes = append(notes, e.Type)
		}
		if e.Context != "" {
			notes = append(notes, e.Context)
		}
		if len(e.Aliases) > 0 {
			notes = append(notes, "aliases: "+strings.Join(e.Aliases, ", "))
		}
		if len(notes) > 0 {
			line += "  (" + strings.Join(notes, "; ") + ")"
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		return ""
	}
	sort.Strings(lines)
	return "TERMINOLOGY GLOSSARY (MANDATORY): When any of the following source terms (or their aliases) " +
		"appear, you MUST translate them exactly as specified below, keeping them consistent throughout. " +
		"Respect the given context to disambiguate; do not apply a term outside its context.\n" +
		strings.Join(lines, "\n")
}

// Merge 把 other 的条目并入（按 term 去重，other 优先覆盖）
func (g *Glossary) Merge(other *Glossary) {
	if other == nil {
		return
	}
	idx := map[string]int{}
	for i, e := range g.Entries {
		idx[strings.ToLower(e.Term)] = i
	}
	for _, e := range other.Entries {
		key := strings.ToLower(e.Term)
		if i, ok := idx[key]; ok {
			// 合并目标语言译法
			if g.Entries[i].Translation == nil {
				g.Entries[i].Translation = map[string]string{}
			}
			for code, t := range e.Translation {
				g.Entries[i].Translation[code] = t
			}
			if e.Context != "" {
				g.Entries[i].Context = e.Context
			}
		} else {
			idx[key] = len(g.Entries)
			g.Entries = append(g.Entries, e)
		}
	}
}
