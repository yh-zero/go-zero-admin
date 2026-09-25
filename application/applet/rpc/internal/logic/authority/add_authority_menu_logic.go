package authoritylogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type AddAuthorityMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddAuthorityMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddAuthorityMenuLogic {
	return &AddAuthorityMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AddAuthorityMenuLogic) AddAuthorityMenu(in *pb.AddAuthorityMenuRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := accessutil.RequireRole(tx, in.AuthorityId); err != nil {
			return err
		}
		var all []model.SysBaseMenu
		if err := tx.Find(&all).Error; err != nil {
			return err
		}
		byID := map[int64]model.SysBaseMenu{}
		for _, menu := range all {
			byID[menu.ID] = menu
		}
		chosen := map[int64]bool{}
		if strings.TrimSpace(in.MenuIds) != "" {
			for _, raw := range strings.Split(in.MenuIds, ",") {
				id, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
				if err != nil || id <= 0 {
					return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单ID列表无效")
				}
				ancestors := map[int64]bool{}
				for id != 0 {
					menu, ok := byID[id]
					if !ok {
						return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单不存在")
					}
					if ancestors[id] {
						return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "菜单存在循环，请先修复菜单层级")
					}
					ancestors[id] = true
					chosen[id] = true
					id = menu.ParentId
				}
			}
		}
		if err := tx.Where("sys_authority_authority_id = ?", in.AuthorityId).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
			return err
		}
		rows := make([]model.SysAuthorityMenu, 0, len(chosen))
		ids := make([]int64, 0, len(chosen))
		for id := range chosen {
			rows = append(rows, model.SysAuthorityMenu{AuthorityId: strconv.FormatInt(in.AuthorityId, 10), MenuId: strconv.FormatInt(id, 10)})
			ids = append(ids, id)
		}
		if len(rows) > 0 {
			if err := tx.Create(&rows).Error; err != nil {
				return err
			}
		}
		buttons := tx.Where("authority_id = ?", in.AuthorityId)
		if len(ids) > 0 {
			buttons = buttons.Where("sys_menu_id NOT IN ?", ids)
		}
		if err := buttons.Delete(&model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		// A removed homepage has deterministic frontend fallback; do not keep a stale name.
		var role model.SysAuthority
		if err := tx.First(&role, "authority_id = ?", in.AuthorityId).Error; err != nil {
			return err
		}
		keepHome := false
		for id := range chosen {
			if byID[id].Name == role.DefaultRouter {
				keepHome = true
			}
		}
		if !keepHome {
			return tx.Model(&role).Update("default_router", "").Error
		}
		return nil
	})
	return &pb.NoDataResponse{}, err
}
