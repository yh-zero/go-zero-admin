package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type AppRefreshLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRefreshLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppRefreshLogic {
	return &AppRefreshLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AppRefreshLogic) AppRefresh(in *apppb.AppRefreshRequest) (*apppb.AppRefreshResponse, error) {
	// 简版：refresh token 机制未实现，提示重新登录
	return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
}
