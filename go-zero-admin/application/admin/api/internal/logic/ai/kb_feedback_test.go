package ai

import (
	"context"
	"strings"
	"testing"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/pkg/xerr"
)

func TestKbFeedbackAPIBoundary(t *testing.T) {
	service := &svc.ServiceContext{Validator: svc.NewValidator()}
	for _, req := range []*types.AiTaskFeedbackReq{
		{Id: 0, Feedback: "helpful"},
		{Id: 1, StepIndex: -1, Feedback: "helpful"},
		{Id: 1, StepIndex: 32, Feedback: "helpful"},
		{Id: 1, Feedback: "bad"},
		{Id: 1, Feedback: "helpful", Note: strings.Repeat("中", 334)},
	} {
		_, err := NewSetTaskFeedbackLogic(context.Background(), service).SetTaskFeedback(req)
		code, ok := xerr.AsCodeError(err)
		if !ok || code.GetErrCode() != 400 {
			t.Fatalf("非法评价未被API拒绝: %v", err)
		}
	}
	_, err := NewReviewKbFeedbackLogic(context.Background(), service).ReviewKbFeedback(&types.AiReviewKbFeedbackReq{Id: 1, FeedbackId: 1, ExpectedRevision: 1, Status: "accepted", Review: &types.AiKbFeedbackReview{Reason: "核对", Evidence: []*types.AiKbFeedbackEvidence{{SourceIndex: 1, Quote: strings.Repeat("中", 167)}}}})
	code, ok := xerr.AsCodeError(err)
	if !ok || code.GetErrCode() != 400 {
		t.Fatalf("嵌套摘录字节上限未拒绝: %v", err)
	}
}

func TestKbFeedbackAPIEightSourceBoundaryBeforeActor(t *testing.T) {
	service := &svc.ServiceContext{Validator: svc.NewValidator()}
	for _, tc := range []struct{ source, code int64 }{{8, 403}, {9, 400}} {
		_, err := NewReviewKbFeedbackLogic(context.Background(), service).ReviewKbFeedback(&types.AiReviewKbFeedbackReq{
			Id: 1, FeedbackId: 1, ExpectedRevision: 1, Status: "dismissed",
			Review: &types.AiKbFeedbackReview{Reason: "核对真实来源", Evidence: []*types.AiKbFeedbackEvidence{{SourceIndex: tc.source, Quote: "实际片段"}}},
		})
		code, ok := xerr.AsCodeError(err)
		if !ok || int64(code.GetErrCode()) != tc.code {
			t.Fatalf("source=%d应返回%d，实际%v", tc.source, tc.code, err)
		}
	}
}
