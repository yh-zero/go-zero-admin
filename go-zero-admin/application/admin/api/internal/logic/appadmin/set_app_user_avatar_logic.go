package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type SetAppUserAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetAppUserAvatarLogic(ctx context.Context, s *svc.ServiceContext) *SetAppUserAvatarLogic {
	return &SetAppUserAvatarLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *SetAppUserAvatarLogic) SetAppUserAvatar(req *types.AppSetAvatarReq) (*types.EmptyResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAdminRpc.SetAppUserAvatar(l.ctx, &apppb.AppSetAvatarRequest{Actor: appAdminActor(l.ctx), UserID: req.Id, FileID: req.FileId})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
