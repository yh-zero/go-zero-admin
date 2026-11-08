package aiservicelogic

import (
	"encoding/json"
	"testing"

	"go-zero-admin/application/ai/rpc/internal/model"
)

func TestEvaluationDatasetSelectionAndRawSnapshot(t *testing.T) {
	dataset, err := loadEvaluationDataset("business-acceptance-cases")
	if err != nil {
		t.Fatal(err)
	}
	if len(dataset.Raw) == 0 || dataset.SHA == "" || len(dataset.Cases) != 24 || !json.Valid([]byte(dataset.Raw)) {
		t.Fatal("内置24题须保留原始字节和指纹")
	}
	first := dataset.Cases[0]
	selected, err := selectEvaluationCases(dataset, first.Kb, []string{first.Id})
	if err != nil || len(selected) != 1 || selected[0].Query != first.Query {
		t.Fatal("合法题目应保持原query")
	}
	if _, err := selectEvaluationCases(dataset, first.Kb, []string{first.Id, first.Id}); err == nil {
		t.Fatal("重复题目必须拒绝")
	}
	if _, err := selectEvaluationCases(dataset, "another", []string{first.Id}); err == nil {
		t.Fatal("跨分组题目必须拒绝")
	}
	if _, err := loadEvaluationDataset("../business-cases"); err == nil {
		t.Fatal("内置题集不能读取任意路径")
	}
}

func TestEvaluationCalibratedDatasetPreservesFrozenParent(t *testing.T) {
	parent, err := loadEvaluationDataset("business-acceptance-cases")
	if err != nil {
		t.Fatal(err)
	}
	derived, err := loadEvaluationDataset("business-evidence-cases")
	if err != nil {
		t.Fatal("校准题集应可通过固定目录读取:", err)
	}
	if len(evaluationDatasetKeys) != 4 || evaluationDatasetKeys[0] != "business-evidence-cases" || derived.SHA != "1b51cb93a575876002bd9877c4ffa2ef3ab95f43785bd5ea4a20292c8b6eb5ca" || len(derived.Cases) != len(parent.Cases) {
		t.Fatal("校准题集目录、指纹或数量不符")
	}
	for i := range parent.Cases {
		if parent.Cases[i] != derived.Cases[i] {
			t.Fatal("校准不能改变被测问题")
		}
	}
}

func TestEvaluationBindingsRequireExactReadyOwnedCorpus(t *testing.T) {
	dataset := &evaluationDataset{Documents: []evaluationDocument{{Key: "guide", Kb: "ops", Source: "guide.md", Title: "运维指南", SHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}}
	row := model.EvaluationCorpusDocument{Id: 10, KbId: 5, Title: "运维指南", Version: 1, Status: "ready", EmbeddingModel: "text-embedding-v4", ContentSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	raw, err := buildEvaluationBindings(dataset, "ops", 5, []model.EvaluationCorpusDocument{row})
	if err != nil || !json.Valid([]byte(raw)) {
		t.Fatal("完全匹配应生成绑定")
	}
	for _, tc := range []struct {
		name   string
		change func(*model.EvaluationCorpusDocument)
	}{
		{"hash", func(r *model.EvaluationCorpusDocument) {
			r.ContentSHA = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
		{"state", func(r *model.EvaluationCorpusDocument) { r.Status = "failed" }},
		{"kb", func(r *model.EvaluationCorpusDocument) { r.KbId = 6 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := row
			tc.change(&changed)
			if _, err := buildEvaluationBindings(dataset, "ops", 5, []model.EvaluationCorpusDocument{changed}); err == nil {
				t.Fatal("正文/状态/归属变化不得建立可比绑定")
			}
		})
	}
	if _, err := buildEvaluationBindings(dataset, "ops", 5, []model.EvaluationCorpusDocument{row, row}); err == nil {
		t.Fatal("重复ID或标题不能任意选原文")
	}
}
