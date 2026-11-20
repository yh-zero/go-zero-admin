package appadmin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
	"google.golang.org/grpc"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/client/appadminservice"
	"go-zero-admin/pkg/xerr"
)

type appReviewRPC struct {
	appadminservice.AppAdminService
	calls int
}

func (rpc *appReviewRPC) ReviewAppUser(_ context.Context, in *apppb.AppReviewRequest, _ ...grpc.CallOption) (*apppb.NoDataResponse, error) {
	rpc.calls++
	if in.UserID != 3 || in.Status != "active" || in.ExpectedStatus != "pending" || in.ExpectedReason == nil || *in.ExpectedReason != "" {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	return nil, xerr.NewErrCodeMsg(409, "审核冲突")
}

func TestReviewAppUserHTTPRequiresExplicitSnapshot(t *testing.T) {
	rpc := &appReviewRPC{}
	logic := NewReviewAppUserLogic(context.Background(), &svc.ServiceContext{Validator: svc.NewValidator(), AppAdminRpc: rpc})
	for _, tc := range []struct {
		payload string
		code    uint32
	}{
		{`{"status":"active"}`, 400},
		{`{"status":"active","expected_status":"pending"}`, 400},
		{`{"status":"active","expected_status":"pending","expected_reason":null}`, 400},
		{`{"status":"active","expected_status":"pending","expected_reason":""}`, 409},
	} {
		r := httptest.NewRequest(http.MethodPost, "/v1/sys/app/users/3/review", strings.NewReader(tc.payload))
		r.Header.Set("Content-Type", "application/json")
		var req types.AppReviewReq
		err := httpx.Parse(pathvar.WithVars(r, map[string]string{"id": "3"}), &req)
		if err == nil {
			_, err = logic.ReviewAppUser(&req)
		}
		if err == nil {
			t.Fatal("预期审核请求被拒绝")
		}
		_, reply := xerr.Handler(r.Context(), err)
		if got := reply.(map[string]any)["code"]; got != tc.code {
			t.Fatalf("请求 %s 应返回 %d，得到 %v", tc.payload, tc.code, got)
		}
	}
	if rpc.calls != 1 {
		t.Fatalf("应只转发携带完整快照的请求，收到 %d 次调用", rpc.calls)
	}
}
