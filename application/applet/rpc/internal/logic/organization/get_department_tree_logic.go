package organizationlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDepartmentTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDepartmentTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDepartmentTreeLogic {
	return &GetDepartmentTreeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDepartmentTreeLogic) GetDepartmentTree(in *pb.OrganizationListRequest) (*pb.DepartmentListResponse, error) {
	return (store{ctx: l.ctx, db: l.svcCtx.DB.DB}).departments(in)
}
