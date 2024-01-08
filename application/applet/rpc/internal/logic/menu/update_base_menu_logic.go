package menulogic

import (
	"context"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateBaseMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBaseMenuLogic {
	return &UpdateBaseMenuLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据id获取菜单
func (l *UpdateBaseMenuLogic) UpdateBaseMenu(in *pb.UpdateBaseMenuRequest) (*pb.UpdateBaseMenuResponse, error) {
	var oldBaseMenu model.SysBaseMenu
	upDateMap := make(map[string]interface{})
	upDateMap["keep_alive"] = in.SysBaseMenu.Meta.KeepAlive
	upDateMap["close_tab"] = in.SysBaseMenu.Meta.CloseTab
	upDateMap["default_menu"] = in.SysBaseMenu.Meta.DefaultMenu
	upDateMap["parent_id"] = in.SysBaseMenu.ParentId
	upDateMap["path"] = in.SysBaseMenu.Path
	upDateMap["name"] = in.SysBaseMenu.Name
	upDateMap["hidden"] = in.SysBaseMenu.Hidden
	upDateMap["component"] = in.SysBaseMenu.Component
	upDateMap["title"] = in.SysBaseMenu.Meta.Title
	upDateMap["active_name"] = in.SysBaseMenu.Meta.ActiveName
	upDateMap["icon"] = in.SysBaseMenu.Meta.Icon
	upDateMap["sort"] = in.SysBaseMenu.Sort

	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		db := tx.Where("id = ?", in.SysBaseMenu.ID).Find(&oldBaseMenu)
		if oldBaseMenu.Name != in.SysBaseMenu.Name {
			if !errors.Is(tx.Where("id <> ? AND name = ?", in.SysBaseMenu.ID, in.SysBaseMenu.Name).First(&model.SysBaseMenu{}).Error, gorm.ErrRecordNotFound) {
				logx.Errorf("存在相同name修改失败")
				return errors.New("存在相同name修改失败")
			}
		}
		txErr := tx.Unscoped().Delete(&model.SysBaseMenuParameter{}, "sys_base_menu_id = ?", in.SysBaseMenu.ID).Error
		if txErr != nil {
			logx.Errorf(" txErr: %v", txErr)
			return txErr
		}
		txErr = tx.Unscoped().Delete(&model.SysBaseMenuBtn{}, "sys_base_menu_id = ?", in.SysBaseMenu.ID).Error
		if txErr != nil {
			logx.Errorf(" txErr: %v", txErr)
			return txErr
		}
		if len(in.SysBaseMenu.Parameters) > 0 {
			for k := range in.SysBaseMenu.Parameters {
				in.SysBaseMenu.Parameters[k].SysBaseMenuID = in.SysBaseMenu.ID
			}
			txErr = tx.Create(&in.SysBaseMenu.Parameters).Error
			if txErr != nil {
				logx.Errorf(" txErr: %v", txErr)
				return txErr
			}
		}
		if len(in.SysBaseMenu.MenuBtn) > 0 {
			for k := range in.SysBaseMenu.MenuBtn {
				in.SysBaseMenu.MenuBtn[k].SysBaseMenuID = in.SysBaseMenu.ID
			}
			txErr = tx.Create(&in.SysBaseMenu.MenuBtn).Error
			if txErr != nil {
				logx.Errorf(" txErr: %v", txErr)
				return txErr
			}
		}

		txErr = db.Updates(upDateMap).Error
		if txErr != nil {
			logx.Errorf(" txErr: %v", txErr)
			return txErr
		}
		return nil
	})

	return &pb.UpdateBaseMenuResponse{}, err
}
