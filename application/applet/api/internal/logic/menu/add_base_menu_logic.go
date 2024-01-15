package menu

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type AddBaseMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAddBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddBaseMenuLogic {
	return &AddBaseMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AddBaseMenuLogic) AddBaseMenu(req *types.AddBaseMenuRequest) (resp *types.AddBaseMenuResponse, err error) {

	var pbSysBaseMenu pb.SysBaseMenu
	_ = copier.Copy(&pbSysBaseMenu, req)

	_, err = l.svcCtx.AppletMenuRPC.AddMenuBase(l.ctx, &pb.AddMenuBaseRequest{SysBaseMenu: &pbSysBaseMenu})
	return &types.AddBaseMenuResponse{}, err
}
