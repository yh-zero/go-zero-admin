package userlogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strconv"
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
	if in == nil {
		return nil, userError("请求不能为空")
	}
	if err := hash.ValidatePassword(in.NewPassword); err != nil {
		return nil, userError(err.Error())
	}
	if in.OldPassword == in.NewPassword {
		return nil, userError("新密码不能与旧密码相同")
	}
	passwordHash, err := hash.BcryptHash(in.NewPassword)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
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
		if err := tx.Model(&user).Updates(map[string]any{"password": passwordHash, "session_version": gorm.Expr("session_version + 1")}).Error; err != nil {
			return err
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "user", Action: "changePassword", Object: strconv.FormatInt(user.ID, 10)})
	})
	return &pb.NoDataResponse{}, err
}
