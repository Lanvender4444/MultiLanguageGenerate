package language

import _ "embed"

// defaultEmbeddedData 是随包内嵌的语言表，作为 LoadEmbedded 的兜底数据源。
// 这样 CLI、公共库等非 GUI 入口无需调用 RegisterEmbeddedData 也能拿到完整语言列表。
//
//go:embed MultiLanguage.json
var defaultEmbeddedData []byte
