package userlogic

import (
	"context"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateUserAuthoritiesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserAuthoritiesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserAuthoritiesLogic {
	return &UpdateUserAuthoritiesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 修改用户和角色的关系信息 -- 和上  在修改用户信息的时候请求
func (l *UpdateUserAuthoritiesLogic) UpdateUserAuthorities(in *pb.UpdateUserAuthoritiesRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		// 删除当前用户的所有权限
		if txErr := tx.Where("sys_user_id = ?", in.ID).Delete(&model.SysUserAuthority{}).Error; txErr != nil {
			return txErr
		}

		// 批量创建新的用户权限记录
		var modelSysUserAuthority []model.SysUserAuthority
		for _, v := range in.AuthorityIds {
			modelSysUserAuthority = append(modelSysUserAuthority, model.SysUserAuthority{SysAuthorityAuthorityId: v, SysUserId: in.ID})
		}
		if txErr := tx.Create(&modelSysUserAuthority).Error; txErr != nil {
			return txErr
		}

		// 更新用户的权限ID字段
		if txErr := tx.Model(&model.SysUser{}).Where("id = ?", in.ID).Update("authority_id", in.AuthorityIds[0]).Error; txErr != nil {
			return txErr
		}

		return nil // 返回 nil 提交事务
	})

	return &pb.NoDataResponse{}, err
}

//func (l *UpdateUserAuthoritiesLogic) UpdateUserAuthorities(in *pb.UpdateUserAuthoritiesRequest) (*pb.UpdateUserAuthoritiesResponse, error) {
//	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
//		txErr := tx.Delete(&model.SysUserAuthority{}, "sys_user_id = ?", in.ID).Error
//		if txErr != nil {
//			return txErr
//		}
//		var modelSysUserAuthority []model.SysUserAuthority
//		for _, v := range in.AuthorityIds {
//			modelSysUserAuthority = append(modelSysUserAuthority, model.SysUserAuthority{SysAuthorityAuthorityId: v, SysUserId: in.ID})
//		}
//		txErr = tx.Create(&modelSysUserAuthority).Error
//		if txErr != nil {
//			return txErr
//		}
//		txErr = tx.Where("id = ?", in.ID).First(&model.SysUser{}).Update("authority_id", in.AuthorityIds[0]).Error
//		if txErr != nil {
//			return txErr
//		}
//		// 返回 nil 提交事务
//		return nil
//	})
//
//	return &pb.UpdateUserAuthoritiesResponse{}, err
//}
