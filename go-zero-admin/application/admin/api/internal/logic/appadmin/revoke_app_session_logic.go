package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type RevokeAppSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeAppSessionLogic(ctx context.Context, s *svc.ServiceContext) *RevokeAppSessionLogic {
	return &RevokeAppSessionLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *RevokeAppSessionLogic) RevokeAppSession(req *types.AppAdminRevokeReq) (*types.EmptyResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAdminRpc.RevokeAppSession(l.ctx, &apppb.AppAdminRevokeRequest{Actor: appAdminActor(l.ctx), UserId: req.Id, SessionID: req.SessionId})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
