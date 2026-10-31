package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateMenuLogic {
	return &UpdateMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateMenuLogic) UpdateMenu(in *adminpb.UpdateMenuRequest) (*adminpb.NoDataResponse, error) {
	// 走生成 model：Update 自动清缓存；FindOne 后校验软删，避免更新已删除行
	menu, err := l.svcCtx.SysMenuModel.FindOne(l.ctx, uint64(in.MenuId))
	if err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if menu.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
	}
	// 防环：新父节点不能是自身或自身后代，否则子树收集会成环
	if in.ParentId > 0 {
		subtree, err := l.svcCtx.SysMenuModel.CollectSubtreeIds(l.ctx, uint64(in.MenuId))
		if err != nil {
			l.Errorf("collect menu subtree error: %v", err)
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		for _, id := range subtree {
			if id == in.ParentId {
				return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "不能挂载到自身或子菜单下")
			}
		}
	}
	menu.ParentId = uint64(in.ParentId)
	menu.Title = in.Title
	menu.Name = in.Name
	menu.Type = int64(in.Type)
	menu.Path = in.Path
	menu.Component = in.Component
	menu.PermCode = in.PermCode
	menu.Icon = in.Icon
	menu.Sort = int64(in.Sort)
	if in.Visible {
		menu.Visible = 1
	} else {
		menu.Visible = 0
	}
	menu.Status = int64(in.Status)
	if err := l.svcCtx.SysMenuModel.Update(l.ctx, menu); err != nil {
		l.Errorf("update menu error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
