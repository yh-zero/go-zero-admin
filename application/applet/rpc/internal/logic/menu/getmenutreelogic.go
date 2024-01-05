package menulogic

import (
	"context"
	"fmt"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetMenuTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuTreeLogic {
	return &GetMenuTreeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取菜单-路由
func (l *GetMenuTreeLogic) GetMenuTree(in *pb.GetMenuTreeRequest) (*pb.GetMenuTreeResponse, error) {
	menuTree, err := l.getMenuTreeMap(in.AuthorityId)
	var menus []model.SysMenu
	menus = menuTree["0"]
	for i := 0; i < len(menus); i++ {
		err = l.getChildrenList(&menus[i], menuTree)
	}
	var menusRes pb.GetMenuTreeResponse
	_ = copier.Copy(&menusRes.SysMenu, menus)
	return &menusRes, err
}

// 获取路由总树
func (l *GetMenuTreeLogic) getMenuTreeMap(authorityId int64) (treeMap map[string][]model.SysMenu, err error) {
	var allMenus []model.SysMenu
	var baseMenu []model.SysBaseMenu
	var btns []model.SysAuthorityBtn
	treeMap = make(map[string][]model.SysMenu)

	var SysAuthorityMenus []model.SysAuthorityMenu
	if err = l.svcCtx.DB.Where("sys_authority_authority_id = ?", authorityId).Find(&SysAuthorityMenus).Error; err != nil {
		return nil, err
	}

	var MenuIds []string
	for i := range SysAuthorityMenus {
		MenuIds = append(MenuIds, SysAuthorityMenus[i].MenuId)
	}

	if err = l.svcCtx.DB.Where("id in (?)", MenuIds).Order("sort").Preload("Parameters").Find(&baseMenu).Error; err != nil {
		return nil, err
	}

	for i := range baseMenu {
		allMenus = append(allMenus, model.SysMenu{
			SysBaseMenu: baseMenu[i],
			MenuId:      strconv.Itoa(int(baseMenu[i].ID)),
			AuthorityId: authorityId,
			Parameters:  baseMenu[i].Parameters,
		})
	}

	if err = l.svcCtx.DB.Where("authority_id = ?", authorityId).Preload("SysBaseMenuBtn").Find(&btns).Error; err != nil {
		return nil, err
	}
	fmt.Println("btns:", btns)
	fmt.Println("btns:", &btns)
	fmt.Printf("\nbtns:%s", &btns)

	var btnMap = make(map[int64]map[string]int64)
	for _, v := range btns {
		if btnMap[v.SysMenuID] == nil {
			btnMap[v.SysMenuID] = make(map[string]int64)
		}
		btnMap[v.SysMenuID][v.SysBaseMenuBtn.Name] = authorityId
	}
	for _, v := range allMenus {
		v.Btns = btnMap[v.ID]
		treeMap[v.ParentId] = append(treeMap[v.ParentId], v)
	}

	return treeMap, err
}

// 获取路由总树
func (l *GetMenuTreeLogic) getChildrenList(menu *model.SysMenu, treeMap map[string][]model.SysMenu) (err error) {
	menu.Children = treeMap[menu.MenuId]
	for i := 0; i < len(menu.Children); i++ {
		err = l.getChildrenList(&menu.Children[i], treeMap)
	}
	return err
}
