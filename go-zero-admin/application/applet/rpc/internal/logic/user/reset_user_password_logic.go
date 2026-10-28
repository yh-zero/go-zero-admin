package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"strconv"
)

type ResetUserPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetUserPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserPasswordLogic {
	return &ResetUserPasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ResetUserPasswordLogic) ResetUserPassword(in *pb.ResetUserPasswordRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.UserID <= 0 {
		return nil, userError("用户ID无效")
	}
	if err := hash.ValidatePassword(l.svcCtx.Config.Default.UserPassword); err != nil {
		return nil, userError("默认重置密码配置无效：" + err.Error())
	}
	passwordHash, err := hash.BcryptHash(l.svcCtx.Config.Default.UserPassword)
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.SysUser{}).Where("id = ?", in.UserID).Updates(map[string]any{"password": passwordHash, "session_version": gorm.Expr("session_version + 1")})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return userError("用户不存在")
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "user", Action: "resetPassword", Object: strconv.FormatInt(in.UserID, 10)})
	})
	return &pb.NoDataResponse{}, err
}
