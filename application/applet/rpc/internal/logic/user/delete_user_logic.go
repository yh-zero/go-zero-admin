package userlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除用户
func (l *DeleteUserLogic) DeleteUser(in *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", in.UserID).Delete(&model.SysUser{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&[]model.SysUserAuthority{}, "sys_user_id = ?", in.UserID).Error; err != nil {
			return err
		}
		return nil
	})

	return &pb.DeleteUserResponse{}, err
}
