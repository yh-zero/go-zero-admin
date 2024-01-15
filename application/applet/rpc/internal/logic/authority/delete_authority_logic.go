package authoritylogic

import (
	"context"
	"fmt"
	"gorm.io/gorm"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAuthorityLogic {
	return &DeleteAuthorityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除角色
func (l *DeleteAuthorityLogic) DeleteAuthority(in *pb.DeleteAuthorityRequest) (*pb.NoDataResponse, error) {
	var modelSysAuthority model.SysAuthority
	modelSysAuthority.AuthorityId = in.ID
	if errors.Is(l.svcCtx.DB.Debug().Preload("Users").First(&modelSysAuthority).Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("该角色不存在")
	}

	if len(modelSysAuthority.Users) != 0 {
		return nil, errors.New("此角色有用户正在使用禁止删除")
	}

	if !errors.Is(l.svcCtx.DB.Where("authority_id = ?", modelSysAuthority.AuthorityId).First(&model.SysUser{}).Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("此角色有用户正在使用禁止删除")
	}

	if !errors.Is(l.svcCtx.DB.Where("parent_id = ?", modelSysAuthority.AuthorityId).First(&model.SysAuthority{}).Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("此角色存在子角色不允许删除")
	}

	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var err error
		//if err = tx.Preload("SysBaseMenus").Preload("DataAuthorityId").Where("authority_id = ?", modelSysAuthority.AuthorityId).First(modelSysAuthority).Unscoped().Delete(modelSysAuthority).Error; err != nil {
		//	return err
		//}
		// 先查询出需要删除的数据并进行 Preload
		if err = tx.Preload("SysBaseMenus").Preload("DataAuthorityId").Where("authority_id = ?", modelSysAuthority.AuthorityId).First(&modelSysAuthority).Error; err != nil {
			return err
		}

		// 删除查询到的数据
		if err = tx.Unscoped().Delete(&modelSysAuthority).Error; err != nil {
			return err
		}
		for _, menu := range modelSysAuthority.SysBaseMenus {
			fmt.Println("menu=", menu)

		}

		if len(modelSysAuthority.SysBaseMenus) > 0 {
			//f err := tx.Model(&modelSysAuthority).Association("SysBaseMenus").Delete(modelSysAuthority.SysBaseMenus); err != nil {
			if err = tx.Model(&modelSysAuthority).Association("SysBaseMenus").Clear(); err != nil {
				return err
			}
		}
		if len(modelSysAuthority.DataAuthorityId) > 0 {
			if err = tx.Model(modelSysAuthority).Association("DataAuthorityId").Delete(modelSysAuthority.DataAuthorityId); err != nil {
				return err
			}
		}

		if err = tx.Delete(&model.SysUserAuthority{}, "sys_authority_authority_id = ?", modelSysAuthority.AuthorityId).Error; err != nil {
			return err
		}
		if err = tx.Where("authority_id = ?", modelSysAuthority.AuthorityId).Delete(&[]model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}

		err = tx.Delete(&gormadapter.CasbinRule{}, "v0 = ?", strconv.FormatInt(modelSysAuthority.AuthorityId, 10)).Error

		if err != nil {
			return err
		}

		return nil
	})

	return &pb.NoDataResponse{}, err
}
