package casbin

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCasbinDataByApiIdsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCasbinDataByApiIdsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataByApiIdsLogic {
	return &UpdateCasbinDataByApiIdsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCasbinDataByApiIdsLogic) UpdateCasbinDataByApiIds(req *types.UpdateCasbinDataByApiIdsRequest) (resp *types.MessageResponse, err error) {
	var rpcReq pb.UpdateCasbinDataByApiIdsRequest
	_ = copier.Copy(&rpcReq, req)
	_, err = l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds(l.ctx, &rpcReq)
	if err != nil {
		logx.Errorf("l.svcCtx.AppletCasbinRPC.UpdateCasbinDataByApiIds err: %v", err)
		return nil, err
	}

	return &types.MessageResponse{
		Message: "修改成功！",
	}, nil
}
