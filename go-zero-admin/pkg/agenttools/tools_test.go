package agenttools

import (
	"encoding/json"
	"testing"
)

func TestToolSpecsKeepReadOnlyNamesAndStrictSchemas(t *testing.T) {
	if len(Specs) != 3 {
		t.Fatal("unexpected tool count")
	}
	for i, name := range []string{"query_audit_logs", "get_file_status", "list_my_devices"} {
		spec := Specs[i]
		if spec.Name != name || spec.Label == "" || spec.Description == "" || spec.Path == "" {
			t.Fatal("incomplete tool contract", spec)
		}
		var schema struct {
			Type                 string         `json:"type"`
			AdditionalProperties *bool          `json:"additionalProperties"`
			Properties           map[string]any `json:"properties"`
		}
		if err := json.Unmarshal([]byte(spec.Schema), &schema); err != nil {
			t.Fatal(err)
		}
		if schema.Type != "object" || schema.AdditionalProperties == nil || *schema.AdditionalProperties {
			t.Fatal("arguments must be a strict object", spec.Name)
		}
		if _, exists := schema.Properties["userId"]; exists {
			t.Fatal("model cannot choose the acting user")
		}
		if spec.SessionOnly != (name == "list_my_devices") {
			t.Fatal("personal device permission contract changed")
		}
	}
}
