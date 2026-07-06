package openapi

import (
	"encoding/json"
	"testing"

	"forwarder-factory/internal/network"
)

func TestBuildInjectsNetworkEnum(t *testing.T) {
	specBytes, err := Build("http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}

	var spec map[string]interface{}
	if err := json.Unmarshal(specBytes, &spec); err != nil {
		t.Fatal(err)
	}

	components := spec["components"].(map[string]interface{})
	schemas := components["schemas"].(map[string]interface{})
	networkName := schemas["NetworkName"].(map[string]interface{})
	enum, ok := networkName["enum"].([]interface{})
	if !ok || len(enum) == 0 {
		t.Fatalf("NetworkName.enum missing or empty: %#v", networkName["enum"])
	}
	if len(enum) != len(network.All) {
		t.Fatalf("NetworkName enum length %d, want %d", len(enum), len(network.All))
	}

	params := components["parameters"].(map[string]interface{})
	for _, key := range []string{"networkName", "networkPath", "networkQuery", "networkQueryOptional"} {
		p := params[key].(map[string]interface{})
		schema := p["schema"].(map[string]interface{})
		if schema["$ref"] != "#/components/schemas/NetworkName" {
			t.Fatalf("parameter %s should $ref NetworkName", key)
		}
	}

	deploy := schemas["DeployRequest"].(map[string]interface{})
	props := deploy["properties"].(map[string]interface{})
	netField := props["network"].(map[string]interface{})
	if netField["$ref"] != "#/components/schemas/NetworkName" {
		t.Fatal("DeployRequest.network should $ref NetworkName")
	}

	call := schemas["ContractCallRequest"].(map[string]interface{})
	callProps := call["properties"].(map[string]interface{})
	callNet := callProps["network"].(map[string]interface{})
	if callNet["$ref"] != "#/components/schemas/NetworkName" {
		t.Fatal("ContractCallRequest.network should $ref NetworkName")
	}

	monitor := schemas["MonitorNetworkRequest"].(map[string]interface{})
	monitorProps := monitor["properties"].(map[string]interface{})
	monitorNet := monitorProps["network"].(map[string]interface{})
	if monitorNet["$ref"] != "#/components/schemas/NetworkName" {
		t.Fatal("MonitorNetworkRequest.network should $ref NetworkName")
	}
}
