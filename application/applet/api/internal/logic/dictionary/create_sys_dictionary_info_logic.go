package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSysDictionaryInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryInfoLogic {
	return &CreateSysDictionaryInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateSysDictionaryInfoLogic) CreateSysDictionaryInfo(req *types.CreateSysDictionaryInfoRequest) (resp *types.MessageResponse, err error) {
	var pbSysDictionaryInfo pb.SysDictionaryInfo
	_ = copier.Copy(&pbSysDictionaryInfo, req)
	_, err = l.svcCtx.AppletDictionaryRPC.CreateSysDictionaryInfo(l.ctx, &pb.CreateSysDictionaryInfoRequest{SysDictionaryInfo: &pbSysDictionaryInfo})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.CreateSysDictionaryInfo err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "创建成功！"}, nil
}
