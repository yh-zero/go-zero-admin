package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type ListAppUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppUsersLogic(ctx context.Context, s *svc.ServiceContext) *ListAppUsersLogic {
	return &ListAppUsersLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *ListAppUsersLogic) ListAppUsers(req *types.AppAdminListReq) (*types.AppAdminListResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAdminRpc.ListAppUsers(l.ctx, &apppb.AppAdminListRequest{Actor: appAdminActor(l.ctx), Page: &apppb.PageRequest{Page: req.Page, PageSize: req.PageSize}, Status: req.Status, Account: req.Account})
	if err != nil {
		return nil, err
	}
	list := make([]types.AppAdminUserInfo, 0, len(r.List))
	for _, u := range r.List {
		list = append(list, mapAdminAppUser(u))
	}
	return &types.AppAdminListResp{List: list, Total: r.GetPage().GetTotal()}, nil
}

func mapAdminAppUser(user *apppb.AppAdminUserInfo) types.AppAdminUserInfo {
	return types.AppAdminUserInfo{
		Id: user.GetId(), Account: user.GetAccount(), IdentityKind: user.GetIdentityKind(),
		Nickname: user.GetNickname(), AvatarFileId: user.GetAvatarFileId(), AvatarUrl: user.GetAvatarUrl(),
		Status: user.GetStatus(), ReviewReason: user.GetReviewReason(), EmailVerified: user.GetEmailVerified(),
		PhoneVerified: user.GetPhoneVerified(), LastLoginAt: user.GetLastLoginAt(),
		CreatedAt: user.GetCreatedAt(), Remark: user.GetRemark(),
	}
}

func appAdminActor(ctx context.Context) *apppb.Actor {
	a := middleware.ActorFromContext(ctx)
	return &apppb.Actor{UserId: a.UserId, Username: a.Username, DeptId: a.DeptId, IP: a.Ip, UserAgent: a.UserAgent}
}
