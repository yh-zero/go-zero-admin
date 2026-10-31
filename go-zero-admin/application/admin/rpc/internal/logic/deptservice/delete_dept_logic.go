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
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if err := l.svcCtx.SysDeptModel.DeleteDepartment(l.ctx, in.DeptId); err != nil {
		return nil, deptMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
