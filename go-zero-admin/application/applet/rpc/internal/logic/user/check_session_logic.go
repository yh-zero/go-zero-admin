package userlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type CheckSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckSessionLogic {
	return &CheckSessionLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *CheckSessionLogic) CheckSession(in *pb.SessionRequest) (*pb.CheckSessionResponse, error) {
	var count int64
	err := sessionQuery(l.svcCtx.DB.WithContext(l.ctx), in).Count(&count).Error
	if err != nil {
		return nil, err
	}
	return &pb.CheckSessionResponse{Valid: count == 1}, nil
}
