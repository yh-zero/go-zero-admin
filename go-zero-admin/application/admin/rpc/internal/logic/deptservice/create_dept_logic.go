package deptservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateDeptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDeptLogic {
	return &CreateDeptLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *CreateDeptLogic) CreateDept(in *adminpb.CreateDeptRequest) (*adminpb.NoDataResponse, error) {
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	status := in.Status
	if status == 0 {
		status = 1
	}
	err := l.svcCtx.SysDeptModel.CreateDepartment(l.ctx, &model.DeptRow{ParentId: in.ParentId, Name: in.Name, Leader: in.Leader, Phone: in.Phone, Email: in.Email, Sort: in.Sort, Status: status})
	if err != nil {
		return nil, deptMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
func deptMutationError(err error) error {
	if errors.Is(err, model.ErrInvalidMutation) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
	}
	if errors.Is(err, sqlx.ErrNotFound) {
		return xerr.NewErrCode(xerr.DEPT_NOT_FOUND)
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
}
