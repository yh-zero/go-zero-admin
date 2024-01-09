package api

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteApiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiLogic {
	return &DeleteApiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteApiLogic) DeleteApi(req *types.DeleteApiRequest) (resp *types.MessageResponse, err error) {
	var sysApi pb.SysApi
	_ = copier.Copy(&sysApi, req.SysApi)
	_, err = l.svcCtx.AppletAPIRPC.DeleteApi(l.ctx, &pb.DeleteApiRequest{SysApi: &sysApi})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAPIRPC.DeleteApi err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "删除成功"}, nil
}
