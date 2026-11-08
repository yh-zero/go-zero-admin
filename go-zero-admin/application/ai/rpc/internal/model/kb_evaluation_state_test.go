package model

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func evaluationStateFixture(t *testing.T) ([]EvaluationObservation, string) {
	t.Helper()
	rows := []EvaluationObservation{
		{CaseId: "case-one", RequestId: "00000000-0000-4000-8000-000000000001", Status: "waiting", Stage: "waiting", Query: "第一问", List: json.RawMessage(`[]`)},
		{CaseId: "case-two", RequestId: "00000000-0000-4000-8000-000000000002", Status: "waiting", Stage: "waiting", Query: "第二问", List: json.RawMessage(`[]`)},
	}
	raw, err := json.Marshal(rows)
	if err != nil {
		t.Fatal(err)
	}
	return rows, string(raw)
}

func TestEvaluationObservationClaimAndCompletion(t *testing.T) {
	rows, raw := evaluationStateFixture(t)
	next := rows[0]
	next.Status = "running"
	next.Stage = "search"
	claimed, completed, failed, err := replaceEvaluationObservation(raw, 0, next, true)
	if err != nil || completed != 0 || failed != 0 {
		t.Fatalf("领取失败: completed=%d failed=%d err=%v", completed, failed, err)
	}
	actual, readErr := decodeEvaluationObservations(claimed)
	if readErr != nil || len(actual) != 2 {
		t.Fatalf("读取已保存观测失败: count=%d err=%v", len(actual), readErr)
	}
	if actual[0].Status != "running" || actual[1].Status != "waiting" || actual[1].RequestId != rows[1].RequestId {
		t.Fatal("领取应只修改指定waiting项")
	}
	if _, _, _, err = replaceEvaluationObservation(claimed, 0, next, true); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("同项不能被重复领取")
	}
	next.TaskId = 123
	next.Stage = "polling"
	progress, _, _, err := replaceEvaluationObservation(claimed, 0, next, false)
	if err != nil {
		t.Fatal(err)
	}
	next.Status = "succeeded"
	next.Stage = "completed"
	next.Analysis = json.RawMessage(`{"version":1,"type":"kb_analysis"}`)
	done, completed, failed, err := replaceEvaluationObservation(progress, 0, next, false)
	if err != nil || completed != 1 || failed != 0 {
		t.Fatalf("完成计数错误: completed=%d failed=%d err=%v", completed, failed, err)
	}
	if _, _, _, err = replaceEvaluationObservation(done, 0, next, false); !errors.Is(err, ErrEvaluationState) {
		t.Fatal("迟到进度不能改写终态")
	}
}

func TestEvaluationObservationRejectsIdentityAndTaskReplacement(t *testing.T) {
	rows, raw := evaluationStateFixture(t)
	next := rows[0]
	next.Status = "running"
	next.Stage = "search"
	claimed, _, _, err := replaceEvaluationObservation(raw, 0, next, true)
	if err != nil {
		t.Fatal(err)
	}
	next.TaskId = 123
	next.Stage = "polling"
	progress, _, _, err := replaceEvaluationObservation(claimed, 0, next, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		change func(*EvaluationObservation)
	}{
		{"case", func(o *EvaluationObservation) { o.CaseId = "another" }},
		{"uuid", func(o *EvaluationObservation) { o.RequestId = "00000000-0000-4000-8000-000000000003" }},
		{"query", func(o *EvaluationObservation) { o.Query = "替换问题" }},
		{"task", func(o *EvaluationObservation) { o.TaskId = 124 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changed := next
			tc.change(&changed)
			if _, _, _, err := replaceEvaluationObservation(progress, 0, changed, false); !errors.Is(err, ErrEvaluationState) {
				t.Fatal("身份或任务标识不得变化")
			}
		})
	}
}

func TestEvaluationObservationFailureBoundsAndUnknown(t *testing.T) {
	rows, raw := evaluationStateFixture(t)
	next := rows[0]
	next.Status = "running"
	next.Stage = "search"
	claimed, _, _, err := replaceEvaluationObservation(raw, 0, next, true)
	if err != nil {
		t.Fatal(err)
	}
	next.Status = "unknown"
	next.Stage = "unknown"
	next.Error = "执行结果未确认"
	unknown, completed, failed, err := replaceEvaluationObservation(claimed, 0, next, false)
	if err != nil || completed != 0 || failed != 1 {
		t.Fatalf("未知不应算成功: %d %d %v", completed, failed, err)
	}
	actual, readErr := decodeEvaluationObservations(unknown)
	if readErr != nil || len(actual) != 2 {
		t.Fatalf("读取未知观测失败: count=%d err=%v", len(actual), readErr)
	}
	if len(actual[0].Analysis) != 0 || actual[1].Status != "waiting" {
		t.Fatal("未知不能伪造分析或启动后续")
	}
	if _, err = decodeEvaluationObservations(strings.Repeat("x", (1<<20)+1)); err == nil {
		t.Fatal("拒绝超大观测")
	}
	if _, err = decodeEvaluationObservations(`[]`); err == nil {
		t.Fatal("批次至少一个观测")
	}
	next.Status = "succeeded"
	next.Analysis = json.RawMessage(`{`)
	if _, _, _, err = replaceEvaluationObservation(claimed, 0, next, false); err == nil {
		t.Fatal("成功观测须有合法分析JSON和真实任务ID")
	}
}

func TestEvaluationObservationRejectsBlankList(t *testing.T) {
	rows, _ := evaluationStateFixture(t)
	rows[0].List = json.RawMessage("   ")
	if validEvaluationObservation(rows[0]) {
		t.Fatal("空白列表不能作为有效JSON数组")
	}
}

func TestCompactEvaluationJSONKeepsOriginalSnapshotBytes(t *testing.T) {
	_, observations := evaluationStateFixture(t)
	var pretty strings.Builder
	// 通过正常JSON解码再缩进模拟MySQL JSON读回时的额外空白。
	var data any
	if json.Unmarshal([]byte(observations), &data) != nil {
		t.Fatal("fixture invalid")
	}
	raw, _ := json.MarshalIndent(data, "", "  ")
	pretty.Write(raw)
	snapshot := "{\n  \"version\": 1\n}\n"
	row := &KbEvaluationRow{Snapshot: snapshot, Bindings: `{ "documents": [], "inventory": [] }`, Context: `{ "model": { "name": "fixture" } }`, Observations: pretty.String()}
	if compactEvaluationRow(row) != nil || row.Snapshot != snapshot {
		t.Fatal("仅压紧JSON列，不得改snapshot原字节")
	}
	if strings.Contains(row.Observations, "\n") {
		t.Fatal("观测读回应保持有界紧凑JSON")
	}
	if _, err := decodeEvaluationObservations(row.Observations); err != nil {
		t.Fatal(err)
	}
}
