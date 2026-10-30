package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	visible := int64(0)
	if in.Visible {
		visible = 1
	}
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO sys_menu (parent_id, title, name, type, path, component, perm_code, icon, sort, visible, status) VALUES (?,?,?,?,?,?,?,?,?,?,?)",
		in.ParentId, in.Title, in.Name, in.Type, in.Path, in.Component, in.PermCode, in.Icon, in.Sort, visible, in.Status)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
