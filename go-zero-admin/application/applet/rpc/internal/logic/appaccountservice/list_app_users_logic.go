package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppUsersLogic {
	return &ListAppUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppUsersLogic) ListAppUsers(in *pb.AppAdminListRequest) (*pb.AppUserListResponse, error) {
	return listUsers(l.ctx, l.svcCtx, in)
}
