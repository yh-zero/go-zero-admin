package user

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetUserPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetUserPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetUserPasswordLogic {
	return &ResetUserPasswordLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ResetUserPasswordLogic) ResetUserPassword(req *types.ResetUserPasswordRequest) (resp *types.ResetUserPasswordResponse, err error) {
	_, err = l.svcCtx.AppletUserRPC.ResetUserPassword(l.ctx, &pb.ResetUserPasswordRequest{UserID: req.UserID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.ResetUserPassword err: %v", err)
		return nil, err
	}

	return &types.ResetUserPasswordResponse{}, err
}
