package screenplay

import "gopkg.in/yaml.v3"

func ParseYAML(input string) (Document, error) {
	var doc Document
	if err := yaml.Unmarshal([]byte(input), &doc); err != nil {
		return Document{}, err
	}
	return doc, nil
}
