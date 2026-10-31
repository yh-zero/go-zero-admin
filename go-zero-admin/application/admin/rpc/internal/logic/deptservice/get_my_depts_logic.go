package deptservicelogic

import (
	"context"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyDeptsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyDeptsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyDeptsLogic {
	return &GetMyDeptsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetMyDeptsLogic) GetMyDepts(in *adminpb.GetMyDeptsRequest) (*adminpb.GetMyDeptsResponse, error) {
	userId := in.GetActor().GetUserId()
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(userId))
	if err != nil || user.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if user.Status != 1 {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	all, err := l.svcCtx.SysUserModel.IsSuperAdmin(l.ctx, userId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	rows, err := l.svcCtx.SysDeptModel.ListMyDepartments(l.ctx, userId, all)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.DeptInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.DeptInfo{Id: r.Id, ParentId: r.ParentId, Name: r.Name, Leader: r.Leader, Phone: r.Phone, Email: r.Email, Sort: r.Sort, Status: r.Status, Path: r.Path, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetMyDeptsResponse{List: list}, nil
}
