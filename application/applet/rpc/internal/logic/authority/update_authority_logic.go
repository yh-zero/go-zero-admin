package authoritylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthorityLogic {
	return &UpdateAuthorityLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateAuthorityLogic) UpdateAuthority(in *pb.UpdateAuthorityRequest) (*pb.UpdateAuthorityResponse, error) {
	var role model.SysAuthority
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := validateAuthority(tx, in.SysAuthority); err != nil {
			return err
		}
		if err := accessutil.RequireRole(tx, in.SysAuthority.AuthorityId); err != nil {
			return err
		}
		if in.SysAuthority.DefaultRouter != "" {
			var menu model.SysBaseMenu
			if err := tx.Where("name = ?", in.SysAuthority.DefaultRouter).First(&menu).Error; err != nil {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "默认首页菜单不存在")
			}
			var count int64
			if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ? AND sys_base_menu_id = ?", in.SysAuthority.AuthorityId, menu.ID).Count(&count).Error; err != nil {
				return err
			}
			if count == 0 {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "默认首页须先授权给该角色")
			}
		}
		if err := tx.Model(&model.SysAuthority{}).Where("authority_id = ?", in.SysAuthority.AuthorityId).Updates(map[string]interface{}{
			"authority_name": in.SysAuthority.AuthorityName, "parent_id": in.SysAuthority.ParentId, "default_router": in.SysAuthority.DefaultRouter,
		}).Error; err != nil {
			return err
		}
		return tx.First(&role, "authority_id = ?", in.SysAuthority.AuthorityId).Error
	})
	if err != nil {
		return nil, err
	}
	result := &pb.SysAuthority{}
	if err := copier.Copy(result, role); err != nil {
		return nil, err
	}
	return &pb.UpdateAuthorityResponse{SysAuthority: result}, nil
}
