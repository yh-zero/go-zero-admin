package authoritylogic

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAuthorityListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityListLogic {
	return &GetAuthorityListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取角色列表
func (l *GetAuthorityListLogic) GetAuthorityList(in *pb.GetAuthorityListRequest) (*pb.GetAuthorityListResponse, error) {
	offset := int(in.Page.PageSize * (in.Page.PageNo - 1))
	db := l.svcCtx.DB.Model(&model.SysAuthority{})
	var total int64
	if err := db.Where("parent_id = ?", "0").Count(&total).Error; total == 0 || err != nil {
		return nil, err
	}

	var authority []model.SysAuthority
	err := db.Limit(int(in.Page.PageSize)).Offset(offset).Preload("DataAuthorityId").Where("parent_id = ?", "0").Find(&authority).Error
	for i := range authority {
		err = l.findChildrenAuthority(&authority[i])
	}
	var authorityList pb.GetAuthorityListResponse
	authorityList.Total = total
	_ = copier.Copy(&authorityList.SysAuthority, authority)
	for i := range authorityList.SysAuthority {
		err = l.findAuthorityMenusIds(authorityList.SysAuthority[i])
	}

	return &authorityList, err
}

// 查询子角色
func (l *GetAuthorityListLogic) findChildrenAuthority(authority *model.SysAuthority) (err error) {
	err = l.svcCtx.DB.Preload("DataAuthorityId").Where("parent_id = ?", authority.AuthorityId).Find(&authority.Children).Error
	if len(authority.Children) > 0 {
		for k := range authority.Children {
			err = l.findChildrenAuthority(&authority.Children[k])
		}
	}
	return err
}

// 查询角色菜单ids
func (l *GetAuthorityListLogic) findAuthorityMenusIds(sysAuthority *pb.SysAuthority) (err error) {
	var MenuIds []int
	err = l.svcCtx.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", sysAuthority.AuthorityId).Pluck("sys_base_menu_id", &MenuIds).Error
	// 将 []int 转换为字符串
	var strSlice []string
	for _, v := range MenuIds {
		strSlice = append(strSlice, fmt.Sprintf("%d", v))
	}
	sysAuthority.ShowMenuIds = strings.Join(strSlice, ",")
	fmt.Println("---------- MenuIds", MenuIds)
	fmt.Println("---------- sysAuthority.ShowMenuIds", sysAuthority.ShowMenuIds)
	if len(sysAuthority.Children) > 0 {
		for i := range sysAuthority.Children {
			err = l.findAuthorityMenusIds(sysAuthority.Children[i])
		}
	}

	return err
}
