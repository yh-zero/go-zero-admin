package appadminservicelogic

import (
	"context"
	"testing"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type conflictingReviewModel struct {
	model.AppUserModel
	expected, next model.ReviewDecision
}

func (m *conflictingReviewModel) SetReview(_ context.Context, _ uint64, expected, next model.ReviewDecision, _ func(context.Context) error) error {
	m.expected, m.next = expected, next
	return model.ErrAppReviewChanged
}

func TestReviewAppUserRequiresOriginalDecisionAndReportsConflict(t *testing.T) {
	users := &conflictingReviewModel{}
	logic := NewReviewAppUserLogic(context.Background(), &svc.ServiceContext{AppUserModel: users})
	req := &apppb.AppReviewRequest{Actor: &apppb.Actor{UserId: 2}, UserID: 3, Status: "active", ExpectedStatus: "pending"}
	checkCode := func(want uint32) {
		t.Helper()
		_, err := logic.ReviewAppUser(req)
		e, ok := xerr.AsCodeError(err)
		if !ok || e.GetErrCode() != want {
			t.Fatalf("应返回业务码 %d，得到 %v", want, err)
		}
	}
	checkCode(xerr.REQUEST_PARAM_ERROR)
	empty := ""
	req.ExpectedReason = &empty
	req.ExpectedStatus = ""
	checkCode(xerr.REQUEST_PARAM_ERROR)
	req.ExpectedStatus = "pending"
	checkCode(409)
	if users.expected != (model.ReviewDecision{Status: "pending"}) || users.next != (model.ReviewDecision{Status: "active"}) {
		t.Fatalf("审核快照或目标状态传递错误：%+v %+v", users.expected, users.next)
	}
}
