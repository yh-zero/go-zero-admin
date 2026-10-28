package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetRoleDataScopeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRoleDataScopeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleDataScopeLogic {
	return &GetRoleDataScopeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetRoleDataScopeLogic) GetRoleDataScope(in *pb.GetRoleDataScopeRequest) (*pb.RoleDataScopeResponse, error) {
	return (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).dataScope(in)
}
