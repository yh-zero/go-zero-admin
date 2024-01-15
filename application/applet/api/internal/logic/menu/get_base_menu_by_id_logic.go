package menu

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBaseMenuByIdLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBaseMenuByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBaseMenuByIdLogic {
	return &GetBaseMenuByIdLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBaseMenuByIdLogic) GetBaseMenuById(req *types.GetBaseMenuByIdRequest) (resp *types.GetBaseMenuByIdResponse, err error) {
	baseMenuById, err := l.svcCtx.AppletMenuRPC.GetBaseMenuById(l.ctx, &pb.GetBaseMenuByIdRequest{ID: req.Id})

	if err != nil {
		logx.Errorf("l.svcCtx.AppletMenuRPC.GetBaseMenuById err: %v", err)
		return nil, err
	}
	var typesSysBaseMenu types.SysBaseMenu
	_ = copier.Copy(&typesSysBaseMenu, baseMenuById.SysBaseMenu)
	return &types.GetBaseMenuByIdResponse{SysBaseMenu: typesSysBaseMenu}, err
}
