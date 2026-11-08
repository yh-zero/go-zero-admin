package ai

import (
	"context"
	"strings"
	"testing"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/pkg/xerr"
)

func TestKbEvaluationRejectsInvalidInputBeforeActorOrRPC(t *testing.T) {
	service := &svc.ServiceContext{Validator: svc.NewValidator()}
	valid := types.AiStartKbEvaluationReq{Id: 1, RequestId: "11111111-1111-4111-8111-111111111111", DatasetKey: "business-acceptance-cases", DatasetSha256: strings.Repeat("a", 64), GroupKey: "project_ops", CaseIds: []string{"case-1"}}
	for _, change := range []func(*types.AiStartKbEvaluationReq){
		func(r *types.AiStartKbEvaluationReq) { r.Id = 0 },
		func(r *types.AiStartKbEvaluationReq) { r.RequestId = "invalid" },
		func(r *types.AiStartKbEvaluationReq) { r.DatasetSha256 = strings.Repeat("g", 64) },
		func(r *types.AiStartKbEvaluationReq) { r.CaseIds = nil },
		func(r *types.AiStartKbEvaluationReq) { r.CaseIds = []string{""} },
		func(r *types.AiStartKbEvaluationReq) { r.CaseIds = []string{"same", "same"} },
		func(r *types.AiStartKbEvaluationReq) { r.CaseIds = make([]string, 31) },
		func(r *types.AiStartKbEvaluationReq) { r.DatasetSnapshot = strings.Repeat("中", (256<<10)/3+1) },
		func(r *types.AiStartKbEvaluationReq) { r.DatasetSnapshot = string([]byte{0xff}) },
	} {
		req := valid
		change(&req)
		_, err := NewStartKbEvaluationLogic(context.Background(), service).StartKbEvaluation(&req)
		code, ok := xerr.AsCodeError(err)
		if !ok || code.GetErrCode() != 400 {
			t.Fatalf("非法评测请求须在Actor/RPC之前返回400：%v", err)
		}
	}
	if err := service.Validator.StructCtx(context.Background(), &valid); err != nil {
		t.Fatal(err)
	}
}
