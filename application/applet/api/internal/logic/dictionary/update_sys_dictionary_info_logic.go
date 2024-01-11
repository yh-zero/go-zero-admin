package dictionary

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSysDictionaryInfoLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryInfoLogic {
	return &UpdateSysDictionaryInfoLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateSysDictionaryInfoLogic) UpdateSysDictionaryInfo(req *types.UpdateSysDictionaryInfoRequest) (resp *types.MessageResponse, err error) {
	var pbSysDictionaryInfo pb.SysDictionaryInfo
	_ = copier.Copy(&pbSysDictionaryInfo, req)
	_, err = l.svcCtx.AppletDictionaryRPC.UpdateSysDictionaryInfo(l.ctx, &pb.UpdateSysDictionaryInfoRequest{SysDictionaryInfo: &pbSysDictionaryInfo})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletDictionaryRPC.UpdateSysDictionaryInfo err:%v", err)
		return nil, err
	}
	return &types.MessageResponse{Message: "修改成功！"}, nil
}
