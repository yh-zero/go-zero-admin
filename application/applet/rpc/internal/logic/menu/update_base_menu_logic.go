package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateBaseMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBaseMenuLogic {
	return &UpdateBaseMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateBaseMenuLogic) UpdateBaseMenu(in *pb.UpdateBaseMenuRequest) (*pb.NoDataResponse, error) {
	if in.SysBaseMenu == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单不能为空")
	}
	err := accessutil.AdminMenuTransaction(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		var old model.SysBaseMenu
		if err := accessutil.RequireID(tx, &old, in.SysBaseMenu.ID); err != nil {
			return err
		}
		if old.Name != in.SysBaseMenu.Name && builtinPermissionMenu(old.Name) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "内置业务菜单标识不可重命名")
		}
		if err := validateMenu(tx, in.SysBaseMenu); err != nil {
			return err
		}
		if old.ParentId != in.SysBaseMenu.ParentId {
			if err := applyMenuMove(tx, old.ID, in.SysBaseMenu.ParentId, in.MovePreviewVersion); err != nil {
				return err
			}
		}
		menu := menuModel(in.SysBaseMenu)
		oldName := old.Name
		fields := map[string]interface{}{"parent_id": menu.ParentId, "path": menu.Path, "name": menu.Name, "hidden": menu.Hidden, "component": menu.Component, "sort": menu.Sort,
			"active_name": menu.ActiveName, "keep_alive": menu.KeepAlive, "default_menu": menu.DefaultMenu, "title": menu.Title, "icon": menu.Icon, "close_tab": menu.CloseTab}
		if err := tx.Model(&old).Updates(fields).Error; err != nil {
			return err
		}
		if oldName != menu.Name {
			if err := tx.Model(&model.SysAuthority{}).Where("default_router = ? AND deleted_at IS NULL", oldName).Update("default_router", menu.Name).Error; err != nil {
				return err
			}
		}
		return saveMenuRelations(tx, old.ID, in.SysBaseMenu)
	})
	return &pb.NoDataResponse{}, accessutil.FriendlyDuplicate(err)
}
