package deptservicelogic

import (
	"context"
	"errors"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDepartmentDirectoryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDepartmentDirectoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDepartmentDirectoryLogic {
	return &GetDepartmentDirectoryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetDepartmentDirectoryLogic) GetDepartmentDirectory(in *adminpb.GetDepartmentDirectoryRequest) (*adminpb.GetDepartmentDirectoryResponse, error) {
	actor := model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()}
	directory, err := l.svcCtx.SysDeptModel.ListDepartmentDirectory(l.ctx, actor, in.GetKeyword(), in.GetPage(), in.GetPageSize())
	switch {
	case errors.Is(err, model.ErrDataScopeDenied):
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	case errors.Is(err, model.ErrInvalidMutation):
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	case err != nil:
		l.Errorf("读取组织通讯录失败: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.DepartmentDirectoryMember, 0, len(directory.List))
	for _, member := range directory.List {
		list = append(list, &adminpb.DepartmentDirectoryMember{Id: member.Id, Name: member.Name, IsLeader: member.IsLeader})
	}
	return &adminpb.GetDepartmentDirectoryResponse{DeptName: directory.DeptName, List: list, Total: directory.Total}, nil
}
