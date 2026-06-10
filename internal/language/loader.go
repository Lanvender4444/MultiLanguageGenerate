package language

import (
	"encoding/json"
	"os"
	"sort"
)

type Language struct {
	Code string
	Name string
}

func Load(path string) ([]Language, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

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
