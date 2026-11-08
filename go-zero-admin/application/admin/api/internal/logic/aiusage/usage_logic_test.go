package aiusage

import (
	"context"
	"testing"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/profileservice"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
)

type usageAPIClient struct {
	aiservice.AiService
	query func(*aipb.UsageQueryRequest)
	set   func(*aipb.SetUsagePolicyRequest)
}

func (s *usageAPIClient) GetUsageSummary(_ context.Context, in *aipb.UsageQueryRequest, _ ...grpc.CallOption) (*aipb.UsageSummaryResponse, error) {
	s.query(in)
	return &aipb.UsageSummaryResponse{Summary: &aipb.UsageSummaryInfo{Period: &aipb.UsagePeriod{Timezone: "Asia/Shanghai"}}}, nil
}
func (s *usageAPIClient) AdminGetUsageSummary(_ context.Context, in *aipb.UsageQueryRequest, _ ...grpc.CallOption) (*aipb.UsageSummaryResponse, error) {
	s.query(in)
	return &aipb.UsageSummaryResponse{}, nil
}
func (s *usageAPIClient) AdminSetUsagePolicy(_ context.Context, in *aipb.SetUsagePolicyRequest, _ ...grpc.CallOption) (*aipb.UsagePolicyResponse, error) {
	s.set(in)
	return &aipb.UsagePolicyResponse{Policy: &aipb.UsagePolicyInfo{}}, nil
}

type usageProfileClient struct {
	profileservice.ProfileService
	super bool
}

func (s *usageProfileClient) GetProfile(_ context.Context, in *adminpb.GetProfileRequest, _ ...grpc.CallOption) (*adminpb.GetProfileResponse, error) {
	return &adminpb.GetProfileResponse{User: &adminpb.UserInfo{Id: in.Actor.UserId, Status: 1, IsSuperAdmin: s.super}}, nil
}
func usageAPIContext() context.Context {
	return context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(7))
}

func TestUsageSelfReadOnlyPassesTrustedActor(t *testing.T) {
	called := false
	client := &usageAPIClient{query: func(in *aipb.UsageQueryRequest) {
		called = true
		if in.Actor.GetUserId() != 7 || in.UserId != 0 {
			t.Fatalf("本人读取不能使用客户端身份：%+v", in)
		}
	}}
	s := &svc.ServiceContext{AiRpc: client, Validator: svc.NewValidator()}
	result, err := NewGetUsageSummaryLogic(usageAPIContext(), s).GetUsageSummary(&types.AiUsageQueryReq{Page: 1, PageSize: 20})
	if err != nil || !called || result == nil || result.Summary == nil {
		t.Fatalf("本人统计未调用真实RPC边界：called=%v err=%v", called, err)
	}
}
func TestUsageAdminReadRequiresCurrentSuperAdmin(t *testing.T) {
	client := &usageAPIClient{query: func(*aipb.UsageQueryRequest) { t.Error("非超管不应查询全用户账本") }}
	s := &svc.ServiceContext{AiRpc: client, ProfileRpc: &usageProfileClient{}, Validator: svc.NewValidator()}
	_, err := NewAdminGetUsageSummaryLogic(usageAPIContext(), s).AdminGetUsageSummary(&types.AiUsageAdminQueryReq{Page: 1, PageSize: 20})
	if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 403 {
		t.Fatalf("应拒绝全用户读取，err=%v", err)
	}
}
func TestUsagePolicyExpectedVersionPresenceAndZero(t *testing.T) {
	called := false
	client := &usageAPIClient{set: func(in *aipb.SetUsagePolicyRequest) {
		called = true
		if in.ExpectedVersion == nil || *in.ExpectedVersion != 0 || in.Actor.UserId != 7 {
			t.Fatalf("新建策略0版本不能被当省略：%+v", in)
		}
	}}
	s := &svc.ServiceContext{AiRpc: client, ProfileRpc: &usageProfileClient{super: true}, Validator: svc.NewValidator()}
	logic := NewAdminSetUsagePolicyLogic(usageAPIContext(), s)
	if _, err := logic.AdminSetUsagePolicy(&types.AiSetUsagePolicyReq{}); err == nil || called {
		t.Fatal("缺expected_version必须在RPC前拒绝")
	}
	zero, limit := int64(0), int64(2)
	if _, err := logic.AdminSetUsagePolicy(&types.AiSetUsagePolicyReq{UserId: 42, ExpectedVersion: &zero, DailyRequestLimit: &limit}); err != nil || !called {
		t.Fatalf("合法0版本新建失败：err=%v", err)
	}
}
