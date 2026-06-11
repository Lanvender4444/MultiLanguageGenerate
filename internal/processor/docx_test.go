package processor

import (
	"strings"
	"testing"
)

func TestExtractParagraphs_Attributes(t *testing.T) {
	xml := `<w:p w14:paraId="123" w14:textId="abc"><w:r><w:t>Hello</w:t></w:r></w:p><w:p><w:r><w:t>World</w:t></w:r></w:p>`

	paras := extractParagraphs(xml)
	if len(paras) != 2 {
		t.Fatalf("expected 2 paragraphs, got %d", len(paras))
	}
	if paras[0] != "Hello" {
		t.Errorf("para0=%q", paras[0])
	}
	if paras[1] != "World" {
		t.Errorf("para1=%q", paras[1])
	}
}

func TestReplaceParagraphsInXML_Attributes(t *testing.T) {
	original := `<w:p w14:paraId="123" w14:textId="abc"><w:r><w:t>Hello</w:t></w:r></w:p><w:p><w:r><w:t>World</w:t></w:r></w:p>`

	result, err := replaceParagraphsInXML(original, []string{"你好", "世界"})
	if err != nil {
		t.Fatal(err)
	}

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

func TestReplaceParagraphsInXML_XMLSpace(t *testing.T) {
	original := `<w:p><w:r><w:t xml:space="preserve">  Hello  </w:t></w:r></w:p>`

	result, err := replaceParagraphsInXML(original, []string{"你好"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `xml:space="preserve"`) {
		t.Errorf("xml:space attribute lost: %s", result)
	}
}

func TestReplaceParagraphsInXML_MoreParasThanTranslations(t *testing.T) {
	original := `<w:p><w:r><w:t>A</w:t></w:r></w:p><w:p><w:r><w:t>B</w:t></w:r></w:p><w:p><w:r><w:t>C</w:t></w:r></w:p>`

	result, err := replaceParagraphsInXML(original, []string{"X"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `<w:t>X</w:t>`) {
		t.Errorf("first para not translated: %s", result)
	}
	if !strings.Contains(result, `<w:t>B</w:t>`) {
		t.Errorf("second para should be preserved: %s", result)
	}
	if !strings.Contains(result, `<w:t>C</w:t>`) {
		t.Errorf("third para should be preserved: %s", result)
	}
}

func TestExtractParagraphs_NoTextParas(t *testing.T) {
	xml := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr></w:p><w:p><w:r><w:t>Content</w:t></w:r></w:p>`

	paras := extractParagraphs(xml)
	if len(paras) != 1 {
		t.Fatalf("expected 1 paragraph (empty paras skipped), got %d", len(paras))
	}
	if paras[0] != "Content" {
		t.Errorf("para0=%q", paras[0])
	}
}

func TestReplaceParagraphs_PreservesNonTextPara(t *testing.T) {
	original := `<w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr></w:p><w:p><w:r><w:t>Content</w:t></w:r></w:p>`

	result, err := replaceParagraphsInXML(original, []string{"内容"})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, `<w:pPr><w:pStyle w:val="Heading1"/></w:pPr>`) {
		t.Errorf("non-text paragraph mangled: %s", result)
	}
}