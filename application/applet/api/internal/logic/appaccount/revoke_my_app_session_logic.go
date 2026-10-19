// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appaccount

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeMyAppSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销本人设备
func NewRevokeMyAppSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMyAppSessionLogic {
	return &RevokeMyAppSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeMyAppSessionLogic) RevokeMyAppSession(req *types.AppSessionIDRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.RevokeMyAppSession(l.ctx, &pb.AppRevokeSelfRequest{AccessToken: appbridge.Request(l.ctx).Token, ID: req.ID})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "操作成功"}, nil
}
