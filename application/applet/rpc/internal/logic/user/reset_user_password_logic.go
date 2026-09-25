package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
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
	if in.UserID <= 0 {
		return nil, userError("用户ID无效")
	}
	result := l.svcCtx.DB.WithContext(l.ctx).Model(&model.SysUser{}).Where("id = ?", in.UserID).Updates(map[string]any{"password": hash.BcryptHash(l.svcCtx.Config.Default.UserPassword), "session_version": gorm.Expr("session_version + 1")})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, userError("用户不存在")
	}
	return &pb.NoDataResponse{}, nil
}
