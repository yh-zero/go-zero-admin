package menulogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBaseMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBaseMenuLogic {
	return &DeleteBaseMenuLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除系统菜单
func (l *DeleteBaseMenuLogic) DeleteBaseMenu(in *pb.DeleteBaseMenuRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Preload("MenuBtn").Preload("Parameters").Where("parent_id = ?", in.ID).First(&model.SysBaseMenu{}).Error
	if err != nil {
		var menu model.SysBaseMenu
		db := l.svcCtx.DB.Preload("SysAuthoritys").Where("id = ?", in.ID).First(&menu).Delete(&menu)
		err = l.svcCtx.DB.Delete(&model.SysBaseMenuParameter{}, "sys_base_menu_id = ?", in.ID).Error
		err = l.svcCtx.DB.Delete(&model.SysBaseMenuBtn{}, "sys_base_menu_id = ?", in.ID).Error
		err = l.svcCtx.DB.Delete(&model.SysAuthorityBtn{}, "sys_menu_id = ?", in.ID).Error
		if err != nil {
			return nil, err
		}
		if len(menu.SysAuthoritys) > 0 {
			err = l.svcCtx.DB.Model(&menu).Association("SysAuthoritys").Delete(&menu.SysAuthoritys)
		} else {
			err = db.Error
			if err != nil {
				return nil, err
			}
		}
	} else {
		return nil, errors.New("此菜单存在子菜单不可删除")
	}
	return &pb.NoDataResponse{}, err
}
