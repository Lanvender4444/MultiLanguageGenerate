package processor

import (
	"strings"
	"testing"
)

func TestParseParagraphs_Attributes(t *testing.T) {
	xml := `<w:p w14:paraId="123" w14:textId="abc"><w:r><w:t>Hello</w:t></w:r></w:p><w:p><w:r><w:t>World</w:t></w:r></w:p>`

	paras := parseParagraphs(xml)
	if len(paras) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(paras))
	}
	if paras[0].FullText != "Hello" {
		t.Errorf("para0=%q", paras[0].FullText)
	}
	if paras[1].FullText != "World" {
		t.Errorf("para1=%q", paras[1].FullText)
	}
	if !strings.HasPrefix(paras[0].OpenTag, `<w:p w14:`) {
		t.Errorf("attributes lost: openTag=%q", paras[0].OpenTag)
	}
}

func TestRebuildXML_Attributes(t *testing.T) {
	original := `<w:p w14:paraId="123" w14:textId="abc"><w:r><w:t>Hello</w:t></w:r></w:p><w:p><w:r><w:t>World</w:t></w:r></w:p>`

	paras := parseParagraphs(original)
	paraContents := []string{
		`<w:r><w:t>你好</w:t></w:r>`,
		`<w:r><w:t>世界</w:t></w:r>`,
	}

	result := rebuildXML(original, paras, paraContents)

	if !strings.Contains(result, `<w:p w14:paraId="123" w14:textId="abc">`) {
		t.Errorf("attribute mangled in output: %s", result)
	}
	if !strings.Contains(result, `<w:t>你好</w:t>`) {
		t.Errorf("translation not found in output: %s", result)
	}
	if !strings.Contains(result, `<w:t>世界</w:t>`) {
		t.Errorf("translation not found in output: %s", result)
	}
	if strings.Contains(result, `<w:pw14:`) {
		t.Errorf("QName corruption detected: %s", result)
	}
}

func TestRebuildParagraphContent_XMLSpace(t *testing.T) {
	xml := `<w:p><w:r><w:t xml:space="preserve">  Hello  </w:t></w:r></w:p>`
	paras := parseParagraphs(xml)

	result := rebuildParagraphContent(paras[0], "你好")
	if !strings.Contains(result, `xml:space="preserve"`) {
		t.Errorf("xml:space attribute lost: %s", result)
	}
}

func TestParseParagraphs_NoTextParas(t *testing.T) {
	xml := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr></w:p><w:p><w:r><w:t>Content</w:t></w:r></w:p>`

	paras := parseParagraphs(xml)
	if len(paras) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(paras))
	}
	if paras[0].HasText {
		t.Errorf("empty paragraph should have HasText=false")
	}
	if !paras[1].HasText {
		t.Errorf("paragraph with content should have HasText=true")
	}
	if paras[1].FullText != "Content" {
		t.Errorf("para1=%q", paras[1].FullText)
	}
}

func TestRebuildXML_NonTextParagraph(t *testing.T) {
	original := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr></w:p><w:p><w:r><w:t>Content</w:t></w:r></w:p>`

	paras := parseParagraphs(original)
	paraContents := make([]string, len(paras))
	paraContents[0] = paras[0].ParaContent
	paraContents[1] = `<w:r><w:t>内容</w:t></w:r>`

	result := rebuildXML(original, paras, paraContents)

	if !strings.Contains(result, `<w:pStyle w:val="Heading1"/>`) {
		t.Errorf("non-text paragraph mangled: %s", result)
	}
	if !strings.Contains(result, `<w:t>内容</w:t>`) {
		t.Errorf("text paragraph not translated: %s", result)
	}
}

func TestEncodeDecodeXMLEntities(t *testing.T) {
	tests := []struct {
		input     string
		encoded   string
	}{
		{"Hello & World", "Hello &amp; World"},
		{"a < b > c", "a &lt; b &gt; c"},
		{`"quoted"`, "&quot;quoted&quot;"},
	}

	for _, tt := range tests {
		enc := encodeXMLEntities(tt.input)
		if enc != tt.encoded {
			t.Errorf("encode(%q)=%q, want %q", tt.input, enc, tt.encoded)
		}
		dec := decodeXMLEntities(enc)
		if dec != tt.input {
			t.Errorf("decode(encode(%q))=%q, want %q", tt.input, dec, tt.input)
		}
	}
}
