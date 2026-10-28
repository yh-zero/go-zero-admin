// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppAuthOptionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询邮箱验证配置
func NewGetAppAuthOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppAuthOptionsLogic {
	return &GetAppAuthOptionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAppAuthOptionsLogic) GetAppAuthOptions() (resp *types.AppAuthOptions, err error) {
	reply, err := l.svcCtx.AppAccountRPC.GetAppAuthOptions(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppAuthOptions](reply)
}
