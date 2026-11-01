package deptservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
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
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	err := l.svcCtx.SysDeptModel.UpdateDepartment(l.ctx, in.DeptId, model.DeptPatch{
		ParentId: in.ParentId, Name: in.Name, Leader: in.Leader, LeaderUserId: in.LeaderUserId,
		Phone: in.Phone, Email: in.Email, Sort: in.Sort, Status: in.Status,
	}, model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()})
	if err != nil {
		return nil, deptMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
