// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppUsersLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询 App 用户与待审核账号
func NewListAppUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppUsersLogic {
	return &ListAppUsersLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppUsersLogic) ListAppUsers(req *types.AppUserListRequest) (resp *types.AppUserList, err error) {
	page := appbridge.Page(req.PageNo, req.PageSize, req.Keyword)
	reply, err := l.svcCtx.AppAccountRPC.ListAppUsers(l.ctx, &pb.AppAdminListRequest{Actor: appbridge.Actor(l.ctx), Page: page, Status: req.Status})
	if err != nil {
		return nil, err
	}
	response, err := appbridge.Copy[types.AppUserList](reply)
	if err != nil {
		return nil, err
	}
	if response.List == nil {
		response.List = []types.AppUser{}
	}
	response.PageNo, response.PageSize = page.PageNo, page.PageSize
	return response, nil
}
