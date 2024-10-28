package convert

import (
	"encoding/json"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// ToJSON converts any type to JSON []byte
func ToJSON(stc interface{}, ident ...bool) []byte {
	if stc == nil {
		return []byte("")
	}
	if stc == "" {
		return []byte("")
	}

	var JSON []byte
	var err error
	if len(ident) > 0 && ident[0] {
		JSON, err = json.MarshalIndent(stc, "", "  ")
		if err != nil {
			slog.Error("error marshalling to JSON", "error", err)
			os.Exit(1)
		}
	} else {
		JSON, err = json.Marshal(stc)
		if err != nil {
			slog.Error("error marshalling to JSON", "error", err)
			os.Exit(1)
		}
	}
	return JSON
}

// ToJSONString converts any type to JSON string
func ToJSONString(stc interface{}, ident ...bool) string {
	return string(ToJSON(stc, ident...))
}

// ToYAML converts any type to YAML []byte
func ToYAML(stc interface{}) []byte {
	if stc == nil {
		return []byte("")
	}
	if stc == "" {
		return []byte("")
	}

	YAML, err := yaml.Marshal(stc)
	if err != nil {
		slog.Error("error marshalling to YAML", "error", err)
		os.Exit(1)
	}
	return YAML
}
