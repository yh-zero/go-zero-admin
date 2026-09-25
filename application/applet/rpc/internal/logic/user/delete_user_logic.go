package userlogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *DeleteUserLogic) DeleteUser(in *pb.DeleteUserRequest) (*pb.NoDataResponse, error) {
	if in.UserID <= 0 {
		return nil, userError("用户ID无效")
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var user model.SysUser
		if err := tx.First(&user, in.UserID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return userError("用户不存在")
			}
			return err
		}
		if err := tx.Delete(&user).Error; err != nil {
			return err
		}
		if err := tx.Where("sys_user_id = ?", in.UserID).Delete(&model.SysUserAuthority{}).Error; err != nil {
			return err
		}
		return accessutil.EnsureUsableAdministrator(tx)
	})
	if err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
