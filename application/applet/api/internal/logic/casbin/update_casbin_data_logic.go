package casbin

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateCasbinDataLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateCasbinDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataLogic {
	return &UpdateCasbinDataLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateCasbinDataLogic) UpdateCasbinData(req *types.UpdateCasbinDataRequest) (resp *types.MessageResponse, err error) {
	var pbUpdateCasbinData pb.UpdateCasbinDataRequest
	_ = copier.Copy(&pbUpdateCasbinData.CasbinInfoList, req.CasbinInfoList)
	pbUpdateCasbinData.AuthorityId = req.AuthorityId
	_, err = l.svcCtx.AppletCasbinRPC.UpdateCasbinData(l.ctx, &pbUpdateCasbinData)
	if err != nil {
		return nil, err
	}

	return &types.MessageResponse{
		Message: "修改成功！",
	}, nil
}
