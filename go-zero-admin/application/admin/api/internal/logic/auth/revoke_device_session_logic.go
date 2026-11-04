// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package auth

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeDeviceSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRevokeDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceSessionLogic {
	return &RevokeDeviceSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeDeviceSessionLogic) RevokeDeviceSession(req *types.RevokeDeviceSessionReq) (resp *types.EmptyResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	_, err = l.svcCtx.AuthRpc.RevokeDeviceSession(l.ctx, &adminpb.RevokeDeviceSessionRequest{Actor: &adminpb.Actor{UserId: userId}, SessionId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
