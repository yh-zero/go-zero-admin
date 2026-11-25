package audit

import "testing"

func TestCSVSafe(t *testing.T) {
	cases := map[string]string{
		"=SUM(A1:A2)": "'=SUM(A1:A2)",
		"+cmd":        "'+cmd",
		"-cmd":        "'-cmd",
		"@cmd":        "'@cmd",
		"  =formula":  "'  =formula",
		"normal":      "normal",
		"/v1/sys":     "/v1/sys",
		"":            "",
	}
	for input, expected := range cases {
		if actual := csvSafe(input); actual != expected {
			t.Errorf("csvSafe(%q)=%q, want %q", input, actual, expected)
		}
	}
}
