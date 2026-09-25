package userlogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ChangePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ChangePasswordLogic) ChangePassword(in *pb.ChangePasswordRequest) (*pb.NoDataResponse, error) {
	if len(in.NewPassword) < 8 || len(in.NewPassword) > 72 {
		return nil, userError("新密码长度必须为8到72字节")
	}
	if in.OldPassword == in.NewPassword {
		return nil, userError("新密码不能与旧密码相同")
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		var user model.SysUser
		err := sessionQuery(tx, in.Session).Clauses(clause.Locking{Strength: "UPDATE"}).First(&user).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return sessionExpired()
		}
		if err != nil {
			return err
		}
		if !hash.BcryptCheck(in.OldPassword, user.Password) {
			return userError("原密码不正确")
		}
		return tx.Model(&user).Updates(map[string]any{"password": hash.BcryptHash(in.NewPassword), "session_version": gorm.Expr("session_version + 1")}).Error
	})
	return &pb.NoDataResponse{}, err
}
