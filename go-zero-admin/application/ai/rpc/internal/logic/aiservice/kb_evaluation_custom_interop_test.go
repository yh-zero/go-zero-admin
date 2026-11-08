package aiservicelogic

import (
	"os"
	"testing"
)

func TestCustomEvaluationFrontendExportPreservesRawBytes(t *testing.T) {
	raw, err := os.ReadFile("testdata/custom-feedback-export-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2255 {
		t.Fatal("前端实际导出的原始2255字节夹具不能被格式化")
	}
	key, sha := customEvaluationIdentity(string(raw))
	dataset, err := parseCustomEvaluationDataset(key, sha, string(raw))
	if err != nil {
		t.Fatal(err)
	}
	if dataset.Raw != string(raw) || dataset.Documents[0].SourceDocId != 9 || dataset.Documents[0].SourceDocVersion != 2 || dataset.Cases[0].Query != "如何发布？" {
		t.Fatal("前端冻结资料ID、版本、问题和原字节须被Go一致接受")
	}
}
