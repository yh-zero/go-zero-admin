package user

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteUserLogic) DeleteUser(req *types.DeleteUserReq) (*types.EmptyResp, error) {
	_, err := l.svcCtx.UserRpc.DeleteUser(l.ctx, &adminpb.DeleteUserRequest{Actor: middleware.ActorFromContext(l.ctx), UserId: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
