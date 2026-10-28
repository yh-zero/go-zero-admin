package dictionary

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSysDictionaryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryLogic {
	return &CreateSysDictionaryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSysDictionaryLogic) CreateSysDictionary(req *types.CreateSysDictionaryRequest) (resp *types.MessageResponse, err error) {
	var pbSysDictionary pb.SysDictionary
	_ = copier.Copy(&pbSysDictionary, req)
	_, err = l.svcCtx.AppletDictionaryRPC.CreateSysDictionary(l.ctx, &pb.CreateSysDictionaryRequest{SysDictionary: &pbSysDictionary})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.CreateSysDictionary err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "创建成功！"}, nil
}
