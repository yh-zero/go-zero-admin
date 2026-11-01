// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAppUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppUserLogic {
	return &GetAppUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAppUserLogic) GetAppUser(req *types.AppAdminUserReq) (resp *types.AppAdminUserResp, err error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	r, err := l.svcCtx.AppAdminRpc.GetAppUser(l.ctx, &apppb.AppAdminUserRequest{Actor: appAdminActor(l.ctx), UserId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.AppAdminUserResp{User: mapAdminAppUser(r.User)}, nil
}
