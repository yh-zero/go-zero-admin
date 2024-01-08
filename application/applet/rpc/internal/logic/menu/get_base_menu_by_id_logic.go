package menulogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

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
	err := l.svcCtx.DB.Preload("MenuBtn").Preload("Parameters").Where("id = ?", in.ID).First(&sysBaseMenu).Error
	var pbSysBaseMenu pb.SysBaseMenu
	_ = copier.Copy(&pbSysBaseMenu, sysBaseMenu)
	return &pb.GetBaseMenuByIdResponse{SysBaseMenu: &pbSysBaseMenu}, err
}
