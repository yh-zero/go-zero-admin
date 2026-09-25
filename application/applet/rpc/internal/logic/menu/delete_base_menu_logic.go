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

type DeleteBaseMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBaseMenuLogic {
	return &DeleteBaseMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteBaseMenuLogic) DeleteBaseMenu(in *pb.DeleteBaseMenuRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var menu model.SysBaseMenu
		if err := accessutil.RequireID(tx, &menu, in.ID); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.SysBaseMenu{}).Where("parent_id = ?", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "此菜单存在子菜单，请先删除子菜单")
		}
		if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_base_menu_id = ?", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "此菜单已分配给角色，请先取消菜单授权")
		}
		if err := tx.Where("sys_menu_id = ?", in.ID).Delete(&model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("sys_base_menu_id = ?", in.ID).Delete(&model.SysBaseMenuParameter{}).Error; err != nil {
			return err
		}
		if err := tx.Unscoped().Where("sys_base_menu_id = ?", in.ID).Delete(&model.SysBaseMenuBtn{}).Error; err != nil {
			return err
		}
		return tx.Delete(&menu).Error
	})
	return &pb.NoDataResponse{}, err
}
