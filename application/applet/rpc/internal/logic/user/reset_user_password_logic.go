package userlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/hash"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetUserPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetUserPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserPasswordLogic {
	return &ResetUserPasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 重置用户密码 默认密码：goZero
func (l *ResetUserPasswordLogic) ResetUserPassword(in *pb.ResetUserPasswordRequest) (*pb.ResetUserPasswordResponse, error) {
	hashedPassword := hash.BcryptHash(l.svcCtx.Config.Default.UserPassword)

	err := l.svcCtx.DB.Model(&model.SysUser{}).Where("id = ?", in.UserID).Update("password", hashedPassword).Error
	return &pb.ResetUserPasswordResponse{}, err
}
