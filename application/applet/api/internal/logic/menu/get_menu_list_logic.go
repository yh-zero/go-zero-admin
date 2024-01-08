package menu

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMenuListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuListLogic {
	return &GetMenuListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMenuListLogic) GetMenuList(req *types.GetMenuListRequest) (resp *types.GetMenuListResponse, err error) {
	//csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher("mysql", l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	//csb := l.svcCtx.Config.CasbinConf.MustNewCasbinWithRedisWatcher(l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.BizRedis)
	//_, err = csb.UpdatePolicy([]string{"100001", "/v1/sys/menu/getMenuList", "GET"}, []string{"100001", "/v1/sys/menu/getMenuList1111", "GET"})
	menuInfoListGet, err := l.svcCtx.AppletMenuRPC.GetMenuBaseInfoList(l.ctx, &pb.GetMenuBaseInfoListRequest{})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletRPC.MenuInfoListGet err:%v", err)
		return nil, err
	}
	var MenuListRes types.GetMenuListResponse
	_ = copier.Copy(&MenuListRes.List, menuInfoListGet.SysBaseMenu)
	MenuListRes.Total = menuInfoListGet.Total
	MenuListRes.PageNo = req.PageNo
	MenuListRes.PageSize = req.PageSize

	return &MenuListRes, err
}
