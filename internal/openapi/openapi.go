package openapi

import (
	_ "embed"
	"encoding/json"

	"forwarder-factory/internal/network"
)

//go:embed base.json
var baseJSON []byte

func Build(serverURL string) ([]byte, error) {
	var spec map[string]interface{}
	if err := json.Unmarshal(baseJSON, &spec); err != nil {
		return nil, err
	}
	spec["servers"] = []map[string]string{{"url": serverURL, "description": "Current server"}}
	injectNetworkEnums(spec, network.Names())
	return json.Marshal(spec)
}

func injectNetworkEnums(spec map[string]interface{}, names []string) {
	components, ok := spec["components"].(map[string]interface{})
	if !ok {
		return
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		return
	}

	networkName := map[string]interface{}{
		"type":        "string",
		"description": "Supported network (select one)",
	}
	if len(names) > 0 {
		networkName["enum"] = toAnySlice(names)
		networkName["example"] = names[0]
	}
	schemas["NetworkName"] = networkName
}

func toAnySlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
