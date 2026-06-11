package language

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

type Language struct {
	Code string
	Name string
}

var embeddedData []byte

func RegisterEmbeddedData(data []byte) {
	embeddedData = data
}

func Load(path string) ([]Language, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return LoadFromBytes(data)
}

func LoadFromBytes(data []byte) ([]Language, error) {
	var m map[string]string
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}

	languages := make([]Language, 0, len(m))
	for code, name := range m {
		languages = append(languages, Language{Code: code, Name: name})
	}

	sort.Slice(languages, func(i, j int) bool {
		return languages[i].Code < languages[j].Code
	})

	return languages, nil
}

func LoadEmbedded() ([]Language, error) {
	if len(embeddedData) == 0 {
		return nil, fmt.Errorf("no embedded language data")
	}
	return LoadFromBytes(embeddedData)
}
