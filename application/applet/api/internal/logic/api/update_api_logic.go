package api

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateApiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateApiLogic {
	return &UpdateApiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateApiLogic) UpdateApi(req *types.UpdateApiRequest) (resp *types.MessageResponse, err error) {
	var pbSysApi pb.SysApi
	_ = copier.Copy(&pbSysApi, req)
	_, err = l.svcCtx.AppletAPIRPC.UpdateApi(l.ctx, &pb.UpdateApiRequest{SysApi: &pbSysApi})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAPIRPC.UpdateApi err:%v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "修改成功！"}, nil
}
