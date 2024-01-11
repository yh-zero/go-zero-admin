package dictionary

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSysDictionaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryLogic {
	return &DeleteSysDictionaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteSysDictionaryLogic) DeleteSysDictionary(req *types.DeleteSysDictionaryRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletDictionaryRPC.DeleteSysDictionary(l.ctx, &pb.DeleteSysDictionaryRequest{ID: req.ID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.DeleteSysDictionary err:%v", err)
		return nil, err
	}
	return &types.MessageResponse{Message: "删除成功！"}, nil
}
