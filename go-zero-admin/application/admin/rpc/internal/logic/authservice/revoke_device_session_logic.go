package authservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type RevokeDeviceSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceSessionLogic {
	return &RevokeDeviceSessionLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RevokeDeviceSessionLogic) RevokeDeviceSession(in *adminpb.RevokeDeviceSessionRequest) (*adminpb.NoDataResponse, error) {
	// 简版：Redis 会话撤销未实现
	return &adminpb.NoDataResponse{}, nil
}
