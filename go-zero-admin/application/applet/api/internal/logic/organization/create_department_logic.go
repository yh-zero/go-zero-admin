// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDepartmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 创建部门
func NewCreateDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDepartmentLogic {
	return &CreateDepartmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateDepartmentLogic) CreateDepartment(req *types.Department) (resp *types.Department, err error) {
	result, err := l.svcCtx.AppletOrganizationRPC.CreateDepartment(l.ctx, &pb.DepartmentRequest{Department: departmentInput(req)})
	if err != nil {
		return nil, err
	}
	return departmentOutput(result), nil
}
