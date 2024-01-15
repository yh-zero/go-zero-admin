package api

import (
	"context"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteApisByIdsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteApisByIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApisByIdsLogic {
	return &DeleteApisByIdsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteApisByIdsLogic) DeleteApisByIds(req *types.DeleteApisByIdsRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletAPIRPC.DeleteApisByIds(l.ctx, &pb.DeleteApisByIdsRequest{Ids: req.Ids})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAPIRPC.DeleteApisByIds err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "删除成功！"}, nil
}
