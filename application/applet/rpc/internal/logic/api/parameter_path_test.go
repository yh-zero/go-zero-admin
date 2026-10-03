package apilogic

import "testing"

func TestSwaggerParametersBecomeCasbinPaths(t *testing.T) {
	data := []byte(`{"swagger":"2.0","paths":{"/v1/ai/runs/{id}":{"get":{"security":[{"BearerAuth":[]}],"x-casbin-resource":true}},"/v1/ai/runs/{id}/cancel":{"post":{"security":[{"BearerAuth":[]}],"x-casbin-resource":true}}}}`)
	rows, err := readSwaggerResources(data)
	if err != nil || len(rows) != 2 {
		t.Fatal(rows, err)
	}
	if rows[0].Path != "/v1/ai/runs/:id" || rows[1].Path != "/v1/ai/runs/:id/cancel" {
		t.Fatal(rows)
	}
}
