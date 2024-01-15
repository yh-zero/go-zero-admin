package dictionary

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSysDictionaryInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryInfoLogic {
	return &DeleteSysDictionaryInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSysDictionaryInfoLogic) DeleteSysDictionaryInfo(req *types.DeleteSysDictionaryInfoRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletDictionaryRPC.DeleteSysDictionaryInfo(l.ctx, &pb.DeleteSysDictionaryInfoRequest{ID: req.ID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.DeleteSysDictionaryInfo err:%v", err)
		return nil, err
	}
	return &types.MessageResponse{Message: "删除成功！"}, nil
}
