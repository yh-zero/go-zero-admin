package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRoleDataScopeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateRoleDataScopeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleDataScopeLogic {
	return &UpdateRoleDataScopeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateRoleDataScopeLogic) UpdateRoleDataScope(in *pb.RoleDataScopeRequest) (*pb.NoDataResponse, error) {
	return &pb.NoDataResponse{}, (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).updateDataScope(in)
}
