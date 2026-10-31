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
	dept, err := l.svcCtx.SysDeptModel.FindOne(l.ctx, uint64(in.DeptId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
	}
	if dept.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
	}
	// 防环：新父节点不能是自身或自身后代（parent_id 被改成环会让子树收集死循环）
	if in.ParentId > 0 {
		subtree, err := l.svcCtx.SysDeptModel.CollectSubtreeIds(l.ctx, uint64(in.DeptId))
		if err != nil {
			l.Errorf("collect dept subtree error: %v", err)
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		for _, id := range subtree {
			if id == in.ParentId {
				return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "不能挂载到自身或子部门下")
			}
		}
		parent, err := l.svcCtx.SysDeptModel.FindOne(l.ctx, uint64(in.ParentId))
		if err != nil || parent.DeletedAt.Valid {
			return nil, xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
		}
	}
	dept.ParentId = uint64(in.ParentId)
	dept.Name = in.Name
	dept.Leader = in.Leader
	dept.Phone = in.Phone
	dept.Email = in.Email
	dept.Sort = int64(in.Sort)
	dept.Status = int64(in.Status)
	// 走生成 model 的 Update，自动失效 id 缓存，避免详情读到脏数据
	if err := l.svcCtx.SysDeptModel.Update(l.ctx, dept); err != nil {
		l.Errorf("update dept error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
