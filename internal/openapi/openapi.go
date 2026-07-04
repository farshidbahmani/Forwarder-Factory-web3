package openapi

import (
	_ "embed"
	"encoding/json"
)

//go:embed base.json
var baseJSON []byte

func Build(serverURL string) ([]byte, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(baseJSON, &spec); err != nil {
		return nil, err
	}
	spec["servers"] = []map[string]string{{"url": serverURL, "description": "Current server"}}
	return json.Marshal(spec)
}
