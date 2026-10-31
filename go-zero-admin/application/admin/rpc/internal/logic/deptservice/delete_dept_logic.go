package deptservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

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
	if err := l.svcCtx.DB.QueryRowCtx(l.ctx, &childCount, "SELECT COUNT(*) FROM sys_dept WHERE parent_id=? AND deleted_at IS NULL", in.DeptId); err != nil {
		l.Errorf("count dept children error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if childCount > 0 {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "存在子部门，无法删除")
	}
	// 校验无用户引用
	var userCount int
	if err := l.svcCtx.DB.QueryRowCtx(l.ctx, &userCount, "SELECT COUNT(*) FROM sys_user WHERE dept_id=? AND deleted_at IS NULL", in.DeptId); err != nil {
		l.Errorf("count dept users error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if userCount > 0 {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "部门下存在用户，无法删除")
	}
	if err := l.svcCtx.SysDeptModel.SoftDelete(l.ctx, uint64(in.DeptId)); err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
		}
		l.Errorf("soft delete dept error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
