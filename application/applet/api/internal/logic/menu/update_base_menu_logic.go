package menu

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateBaseMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateBaseMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateBaseMenuLogic {
	return &UpdateBaseMenuLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateBaseMenuLogic) UpdateBaseMenu(req *types.UpdateBaseMenuRequest) (resp *types.UpdateBaseMenuResponse, err error) {
	var pbSysBaseMenu pb.SysBaseMenu
	_ = copier.Copy(&pbSysBaseMenu, req)
	_, err = l.svcCtx.AppletMenuRPC.UpdateBaseMenu(l.ctx, &pb.UpdateBaseMenuRequest{SysBaseMenu: &pbSysBaseMenu})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletMenuRPC.UpdateBaseMenu err: %v", err)
		return nil, err

	}

	return &types.UpdateBaseMenuResponse{Message: "更新成功！"}, nil
}
