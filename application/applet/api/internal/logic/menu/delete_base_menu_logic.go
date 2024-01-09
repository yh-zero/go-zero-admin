package menu

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteBaseMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteBaseMenuLogic {
	return &DeleteBaseMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteBaseMenuLogic) DeleteBaseMenu(req *types.DeleteBaseMenuRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletMenuRPC.DeleteBaseMenu(l.ctx, &pb.DeleteBaseMenuRequest{ID: req.ID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletMenuRPC.DeleteBaseMenu err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "删除成功！"}, nil
}
