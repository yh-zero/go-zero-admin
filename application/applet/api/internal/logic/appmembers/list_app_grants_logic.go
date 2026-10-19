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

type ListAppGrantsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 分页查询会员授予记录
func NewListAppGrantsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppGrantsLogic {
	return &ListAppGrantsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppGrantsLogic) ListAppGrants(req *types.AppGrantListRequest) (resp *types.AppGrantList, err error) {
	page := appbridge.Page(req.PageNo, req.PageSize, "")
	reply, err := l.svcCtx.AppMembershipRPC.ListAppGrants(l.ctx, &pb.AppGrantListRequest{Actor: appbridge.Actor(l.ctx), Page: page, UserID: req.UserID})
	if err != nil {
		return nil, err
	}
	response, err := appbridge.Copy[types.AppGrantList](reply)
	if err != nil {
		return nil, err
	}
	if response.List == nil {
		response.List = []types.AppGrant{}
	}
	response.PageNo, response.PageSize = page.PageNo, page.PageSize
	return response, nil
}
