package menulogic

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuBaseInfoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuBaseInfoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuBaseInfoListLogic {
	return &GetMenuBaseInfoListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取系统基础菜单列表
func (l *GetMenuBaseInfoListLogic) GetMenuBaseInfoList(in *pb.NoDataResponse) (*pb.GetMenuBaseInfoListResponse, error) {
	var baseMenuList []model.SysBaseMenu
	treeMap, err := l.getBaseMenuTreeMap()
	baseMenuList = treeMap["0"]
	for i := 0; i < len(baseMenuList); i++ {
		err = l.getBaseChildrenList(&baseMenuList[i], treeMap)
	}
	var menuInfoListGetRes pb.GetMenuBaseInfoListResponse
	menuInfoListGetRes.Total = int64(len(baseMenuList))

	_ = copier.Copy(&menuInfoListGetRes.SysBaseMenu, baseMenuList)

	return &menuInfoListGetRes, err
}

// 获取路由总树
func (l *GetMenuBaseInfoListLogic) getBaseMenuTreeMap() (treeMap map[string][]model.SysBaseMenu, err error) {
	var allMenus []model.SysBaseMenu
	treeMap = make(map[string][]model.SysBaseMenu)
	if err = l.svcCtx.DB.Order("sort").Preload("MenuBtn").Preload("Parameters").Find(&allMenus).Error; err != nil {
		return nil, err
	}
	fmt.Println("======= allMenus", allMenus)
	for _, v := range allMenus {
		treeMap[strconv.FormatInt(v.ParentId, 10)] = append(treeMap[strconv.FormatInt(v.ParentId, 10)], v)
	}
	return treeMap, err
}

// 获取菜单的子菜单
func (l *GetMenuBaseInfoListLogic) getBaseChildrenList(menu *model.SysBaseMenu, treeMap map[string][]model.SysBaseMenu) (err error) {
	menu.Children = treeMap[strconv.Itoa(int(menu.ID))]
	for i := 0; i < len(menu.Children); i++ {
		err = l.getBaseChildrenList(&menu.Children[i], treeMap)
	}
	return err
}
