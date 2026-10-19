package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppUserSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppUserSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppUserSessionsLogic {
	return &ListAppUserSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppUserSessionsLogic) ListAppUserSessions(in *pb.AppAdminUserRequest) (*pb.AppSessionListResponse, error) {
	return adminSessions(l.ctx, l.svcCtx, in)
}
