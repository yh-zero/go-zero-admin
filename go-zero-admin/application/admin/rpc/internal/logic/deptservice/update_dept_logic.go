package deptservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateDeptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDeptLogic {
	return &UpdateDeptLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateDeptLogic) UpdateDept(in *adminpb.UpdateDeptRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.SysDeptModel.FindOne(l.ctx, uint64(in.DeptId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
	}
	_, err = l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE sys_dept SET parent_id=?, name=?, leader=?, phone=?, email=?, sort=?, status=? WHERE id=?",
		in.ParentId, in.Name, in.Leader, in.Phone, in.Email, in.Sort, in.Status, in.DeptId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
