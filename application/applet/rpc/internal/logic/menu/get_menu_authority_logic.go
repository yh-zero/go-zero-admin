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

type GetMenuAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuAuthorityLogic {
	return &GetMenuAuthorityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取指定角色menu  -- 用于角色管理的设置权限
func (l *GetMenuAuthorityLogic) GetMenuAuthority(in *pb.GetMenuAuthorityRequest) (*pb.GetMenuAuthorityResponse, error) {
	var baseMenuList []model.SysBaseMenu
	var SysAuthorityMenus []model.SysAuthorityMenu
	err := l.svcCtx.DB.Where("sys_authority_authority_id = ?", in.AuthorityId).Find(&SysAuthorityMenus).Error
	if err != nil {
		return nil, err
	}

	var MenuIds []string
	for i := range SysAuthorityMenus {
		MenuIds = append(MenuIds, SysAuthorityMenus[i].MenuId)
	}

	err = l.svcCtx.DB.Where("id in (?) ", MenuIds).Order("sort").Find(&baseMenuList).Error

	var menus []model.SysMenu
	for i := range baseMenuList {
		menus = append(menus, model.SysMenu{
			SysBaseMenu: baseMenuList[i],
			AuthorityId: in.AuthorityId,
			MenuId:      strconv.FormatInt(baseMenuList[i].ID, 10),
			Parameters:  baseMenuList[i].Parameters,
		})
	}

	var pbGetMenuAuthorityRes pb.GetMenuAuthorityResponse
	var pbSysMenuList []pb.SysMenu
	_ = copier.Copy(&pbSysMenuList, menus)
	_ = copier.Copy(&pbGetMenuAuthorityRes.SysMenuList, pbSysMenuList)

	fmt.Println("----------- menus", menus)
	fmt.Println("----------- pbSysMenuList", pbSysMenuList)
	fmt.Println("----------- pbGetMenuAuthorityRes", pbGetMenuAuthorityRes)

	return &pbGetMenuAuthorityRes, nil
}
