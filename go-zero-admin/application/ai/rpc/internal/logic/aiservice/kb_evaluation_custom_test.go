package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

func customEvaluationFixture(t *testing.T) string {
	t.Helper()
	raw, err := os.ReadFile("testdata/custom-quality-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

type customEvaluationNoRequest struct{ model.AiKbEvaluationModel }

func (*customEvaluationNoRequest) FindRequestOwned(context.Context, int64, string) (*model.KbEvaluationRow, error) {
	return nil, model.ErrNotFound
}

func TestCustomEvaluationRPCValidatesBeforeModelAndReplaysFrozenReceipt(t *testing.T) {
	raw := customEvaluationFixture(t)
	key, sha := customEvaluationIdentity(raw)
	request := &aipb.StartKbEvaluationRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 5, RequestId: uuid.NewString(), DatasetKey: key, DatasetSha256: sha, DatasetSnapshot: raw, GroupKey: "project", CaseIds: []string{"custom-facts"}}
	service := &svc.ServiceContext{AiKbModel: &evaluationOwnedKb{}, AiKbEvaluationModel: &customEvaluationNoRequest{}}
	_, err := NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(request)
	assertWorkspaceCode(t, err, 503)
	signature, err := evaluationRequestSignature(request)
	if err != nil {
		t.Fatal(err)
	}
	receipts := &evaluationReplayStore{row: &model.KbEvaluationRow{Id: 19, OwnerId: 7, KbId: 5, RequestId: request.RequestId, RequestSignature: signature, Status: "failed", Snapshot: raw}}
	service.AiKbEvaluationModel = receipts
	result, err := NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(request)
	if err != nil || result.Evaluation.Id != 19 || result.Evaluation.Snapshot != raw {
		t.Fatal("相同原字节应直接读取原批次，不需要当前模型配置或调用")
	}
	request.DatasetSnapshot += "\n"
	_, err = NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(request)
	assertWorkspaceCode(t, err, 409)
	request.Actor.UserId = 8
	_, err = NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(request)
	assertWorkspaceCode(t, err, 404)
}

func TestCustomEvaluationWorkerUsesFrozenQuestionWithoutGolden(t *testing.T) {
	raw := customEvaluationFixture(t)
	key, sha := customEvaluationIdentity(raw)
	dataset, err := parseCustomEvaluationDataset(key, sha, raw)
	if err != nil {
		t.Fatal(err)
	}
	corpus := []model.EvaluationCorpusDocument{{Id: 10, KbId: 5, Title: "项目说明", Version: 2, Status: "ready", EmbeddingModel: "fixture", ContentSHA: dataset.Documents[0].SHA}}
	bindings, err := buildEvaluationBindings(dataset, "project", 5, corpus)
	if err != nil {
		t.Fatal(err)
	}
	items, _ := json.Marshal([]model.EvaluationObservation{{CaseId: "custom-facts", Query: "预算和保存期是什么？", RequestId: uuid.NewString(), Status: "waiting", Stage: "waiting", List: json.RawMessage(`[]`)}})
	store := &evaluationMemoryStore{row: &model.KbEvaluationRow{Id: 1, OwnerId: 7, KbId: 5, DatasetKey: key, DatasetSHA: sha, GroupKey: "project", Status: "running", CaseCount: 1, Snapshot: raw, Bindings: bindings, Observations: string(items)}, corpus: corpus}
	runtime := &evaluationMemoryRuntime{}
	if err := runEvaluationCases(context.Background(), &aipb.Actor{UserId: 7}, store.row, "lease", store, runtime, 0); err != nil {
		t.Fatal(err)
	}
	if store.row.Status != "succeeded" || !store.row.CorpusVerified || runtime.searches != 1 || runtime.submissions != 1 || runtime.reads != 1 || len(runtime.queries) != 1 || runtime.queries[0] != "预算和保存期是什么？" || store.corpusReads != 2 {
		t.Fatal("自定义题集应复用原worker和单题调用次数，仅发送冻结问题，不送golden或全文")
	}
}

func TestCustomEvaluationCountsRawBytesAndNormalizedInlineEvidence(t *testing.T) {
	raw := customEvaluationFixture(t)
	valid := strings.Replace(raw, `"required_groups":[["五万元"],["30天"]]`, `"required_groups":[["五万元"],["３０ 天"]]`, 1)
	key, sha := customEvaluationIdentity(valid)
	if _, err := parseCustomEvaluationDataset(key, sha, valid); err != nil {
		t.Fatal("NFKC+空白归一应与v1 JS评分一致")
	}
	limit := raw + strings.Repeat(" ", (256<<10)-len(raw))
	key, sha = customEvaluationIdentity(limit)
	if _, err := parseCustomEvaluationDataset(key, sha, limit); err != nil {
		t.Fatal("256KiB合法原字节边界应可冻结")
	}
	key, sha = customEvaluationIdentity(limit + " ")
	if _, err := parseCustomEvaluationDataset(key, sha, limit+" "); err == nil {
		t.Fatal("超过256KiB必须拒绝")
	}
	for _, count := range []int{100, 101} {
		var source map[string]any
		if json.Unmarshal([]byte(raw), &source) != nil {
			t.Fatal("fixture")
		}
		first := source["cases"].([]any)[0].(map[string]any)
		cases := make([]any, count)
		for i := range cases {
			copy := make(map[string]any, len(first))
			for k, v := range first {
				copy[k] = v
			}
			copy["id"] = fmt.Sprintf("case-%d", i)
			cases[i] = copy
		}
		source["cases"] = cases
		bytes, _ := json.Marshal(source)
		key, sha = customEvaluationIdentity(string(bytes))
		dataset, err := parseCustomEvaluationDataset(key, sha, string(bytes))
		if count == 100 && err != nil || count == 101 && err == nil {
			t.Fatalf("题目数量边界未生效: count=%d err=%v", count, err)
		}
		if count == 100 {
			ids := make([]string, 31)
			for i := range ids {
				ids[i] = fmt.Sprintf("case-%d", i)
			}
			if _, err := selectEvaluationCases(dataset, "project", ids); err == nil {
				t.Fatal("每批仍最多30题")
			}
		}
	}
	for _, count := range []int{100, 101} {
		var source map[string]any
		if json.Unmarshal([]byte(raw), &source) != nil {
			t.Fatal("fixture")
		}
		first := source["documents"].([]any)[0].(map[string]any)
		docs := make([]any, count)
		for i := range docs {
			copy := make(map[string]any, len(first))
			for key, value := range first {
				copy[key] = value
			}
			if i > 0 {
				copy["key"] = fmt.Sprintf("guide-%d", i)
				copy["source"] = fmt.Sprintf("inline:guide-%d", i)
				copy["source_doc_id"] = 1000 + i
			}
			docs[i] = copy
		}
		source["documents"] = docs
		bytes, _ := json.Marshal(source)
		key, sha = customEvaluationIdentity(string(bytes))
		_, err := parseCustomEvaluationDataset(key, sha, string(bytes))
		if count == 100 && err != nil || count == 101 && err == nil {
			t.Fatalf("资料数量边界未生效: count=%d err=%v", count, err)
		}
	}
}

func TestCustomEvaluationRejectsFalseIdentityAndAcceptsScopedNegative(t *testing.T) {
	raw := customEvaluationFixture(t)
	key, sha := customEvaluationIdentity(raw)
	for _, identity := range [][2]string{{key, strings.Repeat("a", 64)}, {"business-cases", sha}, {"custom-" + strings.Repeat("b", 32), sha}} {
		if _, err := parseCustomEvaluationDataset(identity[0], identity[1], raw); err == nil {
			t.Fatal("自定义标识和原字节SHA必须全量一致")
		}
	}
	var source map[string]any
	if json.Unmarshal([]byte(raw), &source) != nil {
		t.Fatal("fixture")
	}
	item := source["cases"].([]any)[0].(map[string]any)
	item["query"] = "负责人是谁？"
	item["expected_documents"] = []any{}
	item["evidence"] = []any{}
	item["checks"] = []any{}
	item["expected_answer_example"] = "当前资料不足以确定负责人。"
	item["answer"] = map[string]any{"kind": "unanswerable", "required_groups": []any{}, "forbidden_phrases": []any{}, "absence_policy": "required"}
	bytes, _ := json.Marshal(source)
	key, sha = customEvaluationIdentity(string(bytes))
	if _, err := parseCustomEvaluationDataset(key, sha, string(bytes)); err != nil {
		t.Fatalf("结构合法的资料不足题应保持v1兼容: %v", err)
	}
}

func customEvaluationIdentity(raw string) (string, string) {
	sum := sha256.Sum256([]byte(raw))
	sha := hex.EncodeToString(sum[:])
	return "custom-" + sha[:32], sha
}

func TestCustomEvaluationPreservesRawAndKeepsContentOutOfBindings(t *testing.T) {
	raw := customEvaluationFixture(t)
	key, sha := customEvaluationIdentity(raw)
	dataset, err := parseCustomEvaluationDataset(key, sha, raw)
	if err != nil || dataset == nil || dataset.Raw != raw || dataset.SHA != sha || len(dataset.Cases) != 1 || dataset.Cases[0].Query != "预算和保存期是什么？" || dataset.Documents[0].Content != "预算五万元。保存30天。" {
		t.Fatalf("合法自定义题集须保留原始快照、问题及原文: %v", err)
	}
	row := model.EvaluationCorpusDocument{Id: 10, KbId: 5, Title: "项目说明", Version: 2, Status: "ready", EmbeddingModel: "local-test", ContentSHA: dataset.Documents[0].SHA}
	bindings, err := buildEvaluationBindings(dataset, "project", 5, []model.EvaluationCorpusDocument{row})
	if err != nil || strings.Contains(bindings, `"content":`) || strings.Contains(bindings, "预算五万元") {
		t.Fatal("绑定只能保留指纹与ID，全文只存在冻结快照")
	}
	duplicateTitle := row
	duplicateTitle.Id = 11
	duplicateTitle.ContentSHA = strings.Repeat("b", 64)
	if _, err := buildEvaluationBindings(dataset, "project", 5, []model.EvaluationCorpusDocument{row, duplicateTitle}); err != nil {
		t.Fatal("自定义精确ID绑定不应被另一篇同标题资料替代或干扰")
	}
	for _, change := range []func(*model.EvaluationCorpusDocument){
		func(r *model.EvaluationCorpusDocument) { r.Id = 11 },
		func(r *model.EvaluationCorpusDocument) { r.KbId = 6 },
		func(r *model.EvaluationCorpusDocument) { r.Version = 3 },
		func(r *model.EvaluationCorpusDocument) { r.Status = "pending" },
		func(r *model.EvaluationCorpusDocument) { r.ContentSHA = strings.Repeat("b", 64) },
	} {
		wrong := row
		change(&wrong)
		if _, err := buildEvaluationBindings(dataset, "project", 5, []model.EvaluationCorpusDocument{wrong}); err == nil {
			t.Fatal("跨身份ID、版本、归属、ready或正文变化必须拒绝，不能按标题回退")
		}
	}
}

func TestCustomEvaluationRejectsAliasesDuplicateKeysAndInvalidGolden(t *testing.T) {
	raw := customEvaluationFixture(t)
	for _, mutate := range []func(string) string{
		func(s string) string {
			return strings.Replace(s, `"query":"预算和保存期是什么？"`, `"query":"预算和保存期是什么？","Query":"另一问题"`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"query":"预算和保存期是什么？"`, `"query":"预算和保存期是什么？","que\u0072y":"另一问题"`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"source_doc_version":2`, `"source_doc_version":0`, 1)
		},
		func(s string) string { return strings.Replace(s, `"source_doc_id":10`, `"source_doc_id":-1`, 1) },
		func(s string) string { return strings.Replace(s, `"inline:guide"`, `"other:guide"`, 1) },
		func(s string) string {
			return strings.Replace(s, `"content":"预算五万元。保存30天。"`, `"content":"伪造正文"`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"groups":[["预算五万元"],["保存30天"]]`, `"groups":[["不存在的证据"]]`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"required_groups":[["五万元"],["30天"]]`, `"required_groups":[["不存在的事实"]]`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"expected_documents":["guide"]`, `"expected_documents":["missing"]`, 1)
		},
		func(s string) string { return strings.Replace(s, `"required_group":1`, `"required_group":0`, 1) },
		func(s string) string { return strings.Replace(s, `"kind":"answerable"`, `"kind":"unanswerable"`, 1) },
		func(s string) string {
			return strings.Replace(s, `"query":"预算和保存期是什么？"`, `"query":"\ud800"`, 1)
		},
		func(s string) string {
			return strings.Replace(s, `"feedback_regressions":[]`, `"feedback_regressions":`+strings.Repeat("[", 65)+"0"+strings.Repeat("]", 65), 1)
		},
		func(s string) string { return s + strings.Repeat(" ", 256<<10) },
	} {
		invalid := mutate(raw)
		key, sha := customEvaluationIdentity(invalid)
		if _, err := parseCustomEvaluationDataset(key, sha, invalid); err == nil {
			t.Fatal("非法自定义题集被接受")
		}
	}
}

func TestCustomEvaluationRequestSignatureFreezesRawButKeepsBuiltinIdentity(t *testing.T) {
	raw := customEvaluationFixture(t)
	key, sha := customEvaluationIdentity(raw)
	request := &aipb.StartKbEvaluationRequest{RequestId: uuid.NewString(), KbId: 5, DatasetKey: key, DatasetSha256: sha, GroupKey: "project", CaseIds: []string{"custom-facts"}, DatasetSnapshot: raw}
	first, err := evaluationRequestSignature(request)
	if err != nil {
		t.Fatal(err)
	}
	request.DatasetSnapshot += "\n"
	second, err := evaluationRequestSignature(request)
	if err != nil || first == second {
		t.Fatal("相同UUID的原字节变化必须改变幂等签名")
	}
	request.DatasetSnapshot = ""
	legacy, err := evaluationRequestSignature(request)
	value, _ := json.Marshal(map[string]any{"kb_id": request.KbId, "dataset_key": request.DatasetKey, "dataset_sha256": request.DatasetSha256, "group_key": request.GroupKey, "case_ids": request.CaseIds})
	expected := sha256.Sum256(value)
	if err != nil || legacy != hex.EncodeToString(expected[:]) {
		t.Fatal("未传snapshot必须保持旧签名算法")
	}
}

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
