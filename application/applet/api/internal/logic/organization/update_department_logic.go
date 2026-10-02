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

type UpdateDepartmentLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新部门
func NewUpdateDepartmentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDepartmentLogic {
	return &UpdateDepartmentLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateDepartmentLogic) UpdateDepartment(req *types.Department) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletOrganizationRPC.UpdateDepartment(l.ctx, &pb.DepartmentRequest{Department: departmentInput(req)})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "更新成功"}, nil
}
