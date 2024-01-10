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

type GetBaseMenuTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBaseMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBaseMenuTreeLogic {
	return &GetBaseMenuTreeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户动态路由树  -- 用于角色管理的设置权限
func (l *GetBaseMenuTreeLogic) GetBaseMenuTree(in *pb.NoDataResponse) (*pb.GetBaseMenuTreeResponse, error) {
	treeMap, err := l.getBaseMenuTreeMap()

	var menus []model.SysBaseMenu
	menus = treeMap["0"]
	for i := 0; i < len(menus); i++ {
		err = l.getBaseChildrenList(&menus[i], treeMap)
	}
	var pbSysBaseMenuList []*pb.SysBaseMenu
	_ = copier.Copy(&pbSysBaseMenuList, menus)
	fmt.Println("---------- pbSysBaseMenuList", pbSysBaseMenuList)

	return &pb.GetBaseMenuTreeResponse{
		SysBaseMenuList: pbSysBaseMenuList,
	}, err
}

// 获取基础路由树
func (l *GetBaseMenuTreeLogic) getBaseMenuTreeMap() (treeMap map[string][]model.SysBaseMenu, err error) {
	var sysBaseMenuList []model.SysBaseMenu
	treeMap = make(map[string][]model.SysBaseMenu)
	err = l.svcCtx.DB.Order("sort").Preload("MenuBtn").Preload("Parameters").Find(&sysBaseMenuList).Error
	for _, v := range sysBaseMenuList {
		treeMap[strconv.FormatInt(v.ParentId, 10)] = append(treeMap[strconv.FormatInt(v.ParentId, 10)], v)
	}
	return treeMap, err
}

// 获取菜单的子菜单
func (l *GetBaseMenuTreeLogic) getBaseChildrenList(sysBaseMenu *model.SysBaseMenu, treeMap map[string][]model.SysBaseMenu) (err error) {
	sysBaseMenu.Children = treeMap[strconv.Itoa(int(sysBaseMenu.ID))]
	for i := 0; i < len(sysBaseMenu.Children); i++ {
		err = l.getBaseChildrenList(&sysBaseMenu.Children[i], treeMap)
	}
	return err
}
