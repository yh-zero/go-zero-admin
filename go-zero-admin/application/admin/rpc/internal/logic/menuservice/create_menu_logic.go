package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMenuLogic {
	return &CreateMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateMenuLogic) CreateMenu(in *adminpb.CreateMenuRequest) (*adminpb.NoDataResponse, error) {
	// 保留原有负数父 ID 写入失败的语义，避免无符号转换后插入极大值。
	if in.ParentId < 0 {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	status, err := model.CreateStatus(in.Status)
	if err != nil {
		return nil, menuMutationError(err)
	}
	visible := int64(0)
	if in.Visible {
		visible = 1
	}
	err = l.svcCtx.SysMenuModel.Create(l.ctx, &model.SysMenu{
		ParentId: uint64(in.ParentId), Title: in.Title, Name: in.Name, Type: int64(in.Type),
		Path: in.Path, Component: in.Component, PermCode: in.PermCode, Icon: in.Icon,
		Sort: int64(in.Sort), Visible: visible, Status: status,
	})
	if model.MutationCommitted(err) {
		l.svcCtx.PublishPolicyChanged(l.ctx)
	}
	if err != nil {
		return nil, menuMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
