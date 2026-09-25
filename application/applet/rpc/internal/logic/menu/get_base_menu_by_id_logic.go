package menulogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBaseMenuByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBaseMenuByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBaseMenuByIdLogic {
	return &GetBaseMenuByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据id获取菜单
func (l *GetBaseMenuByIdLogic) GetBaseMenuById(in *pb.GetBaseMenuByIdRequest) (*pb.GetBaseMenuByIdResponse, error) {
	var sysBaseMenu model.SysBaseMenu
	err := accessutil.RequireID(l.svcCtx.DB.Preload("MenuBtn").Preload("Parameters"), &sysBaseMenu, in.ID)
	var pbSysBaseMenu pb.SysBaseMenu
	_ = copier.Copy(&pbSysBaseMenu, sysBaseMenu)
	return &pb.GetBaseMenuByIdResponse{SysBaseMenu: &pbSysBaseMenu}, err
}
