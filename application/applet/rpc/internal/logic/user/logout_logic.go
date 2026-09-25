package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *LogoutLogic) Logout(in *pb.SessionRequest) (*pb.NoDataResponse, error) {
	// Only revoke the version that the caller actually owns.
	err := sessionQuery(l.svcCtx.DB.WithContext(l.ctx), in).Update("session_version", gorm.Expr("session_version + 1")).Error
	return &pb.NoDataResponse{}, err
}
