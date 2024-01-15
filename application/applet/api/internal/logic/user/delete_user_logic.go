package user

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteUserLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserLogic) DeleteUser(req *types.DeleteUserRequest) (resp *types.MessageResponse, err error) {
	jwtUserId := ctxJwt.GetJwtDataID(l.ctx)
	if jwtUserId == req.UserID {
		logx.Errorf("不能自己删除自己")
		return nil, xerr.NewErrCodeMsg(900000, "不能自己删除自己")
	}
	_, err = l.svcCtx.AppletUserRPC.DeleteUser(l.ctx, &pb.DeleteUserRequest{UserID: req.UserID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletUserRPC.DeleteUser err: %v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "删除成功"}, nil
}
