package authservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type ListDeviceSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDeviceSessionsLogic {
	return &ListDeviceSessionsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListDeviceSessionsLogic) ListDeviceSessions(in *adminpb.ListDeviceSessionsRequest) (*adminpb.ListDeviceSessionsResponse, error) {
	// 简版：设备会话基于 Redis，未实现则返回空
	return &adminpb.ListDeviceSessionsResponse{List: []*adminpb.DeviceSessionInfo{}, Page: &adminpb.PageResponse{Total: 0}}, nil
}
