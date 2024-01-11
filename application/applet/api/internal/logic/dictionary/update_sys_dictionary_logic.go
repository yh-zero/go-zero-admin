package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSysDictionaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryLogic {
	return &UpdateSysDictionaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSysDictionaryLogic) UpdateSysDictionary(req *types.UpdateSysDictionaryRequest) (resp *types.MessageResponse, err error) {
	var pbSysDictionary pb.SysDictionary
	_ = copier.Copy(&pbSysDictionary, req)
	_, err = l.svcCtx.AppletDictionaryRPC.UpdateSysDictionary(l.ctx, &pb.UpdateSysDictionaryRequest{SysDictionary: &pbSysDictionary})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.UpdateSysDictionary err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "修改成功！"}, nil
}
