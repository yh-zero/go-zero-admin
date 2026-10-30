package deptservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteDeptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDeptLogic {
	return &DeleteDeptLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteDeptLogic) DeleteDept(in *adminpb.DeleteDeptRequest) (*adminpb.NoDataResponse, error) {
	// 校验无子部门
	var childCount int
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &childCount, "SELECT COUNT(*) FROM sys_dept WHERE parent_id=? AND deleted_at IS NULL", in.DeptId)
	if childCount > 0 {
		return nil, xerr.NewErrCodeMsg(400, "存在子部门，无法删除")
	}
	// 校验无用户引用
	var userCount int
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &userCount, "SELECT COUNT(*) FROM sys_user WHERE dept_id=? AND deleted_at IS NULL", in.DeptId)
	if userCount > 0 {
		return nil, xerr.NewErrCodeMsg(400, "部门下存在用户，无法删除")
	}
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_dept SET deleted_at=NOW() WHERE id=?", in.DeptId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
