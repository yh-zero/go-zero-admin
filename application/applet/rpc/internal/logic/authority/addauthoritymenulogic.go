package authoritylogic

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type AddAuthorityMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddAuthorityMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddAuthorityMenuLogic {
	return &AddAuthorityMenuLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 增加base_menu和角色关联关系 -- 用于角色管理的设置权限
func (l *AddAuthorityMenuLogic) AddAuthorityMenu(in *pb.AddAuthorityMenuRequest) (*pb.AddAuthorityMenuResponse, error) {
	// 开启数据库事务
	tx := l.svcCtx.DB.Begin()

	// 删除特定权限ID（AuthorityId）相关的所有 SysAuthorityMenu 数据
	if err := tx.Where("sys_authority_authority_id = ?", in.AuthorityId).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 查询所有 model.SysBaseMenu{} 表中的 MenuIds
	var allMenuIds []string
	if err := tx.Model(&model.SysBaseMenu{}).Pluck("id", &allMenuIds).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	fmt.Println("---------------- allMenuIds", allMenuIds)

	// 如果存在新的 MenuIds，则检查并插入对应的数据
	if len(in.MenuIds) > 0 {
		menuIds := strings.Split(in.MenuIds, ",")
		var sysAuthorityMenuList []model.SysAuthorityMenu

		for _, v := range menuIds {
			// 检查 MenuId 是否存在于 allMenuIds 中
			if contains(allMenuIds, v) {
				sysAuthorityMenuList = append(sysAuthorityMenuList, model.SysAuthorityMenu{
					MenuId:      v,
					AuthorityId: strconv.FormatInt(in.AuthorityId, 10),
				})
			}
		}

		// 如果有符合条件的数据则批量插入
		if len(sysAuthorityMenuList) > 0 {
			if err := tx.Create(&sysAuthorityMenuList).Error; err != nil {
				tx.Rollback()
				return nil, err
			}
		}
	}

	// 提交事务
	tx.Commit()

	return &pb.AddAuthorityMenuResponse{}, nil
}

// 判断切片中是否包含某个元素
func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

//func (l *AddAuthorityMenuLogic) AddAuthorityMenu(in *pb.AddAuthorityMenuRequest) (*pb.AddAuthorityMenuResponse, error) {
//	// 开启数据库事务
//	tx := l.svcCtx.DB.Begin()
//
//	// 删除特定权限ID（AuthorityId）相关的所有 SysAuthorityMenu 数据
//	if err := tx.Where("sys_authority_authority_id = ?", in.AuthorityId).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
//		tx.Rollback()
//		return nil, err
//	}
//
//	// 如果存在新的 MenuIds，则插入对应的数据
//	if len(in.MenuIds) > 0 {
//		menuIds := strings.Split(in.MenuIds, ",")
//		var sysAuthorityMenuList []model.SysAuthorityMenu
//
//		for _, v := range menuIds {
//			sysAuthorityMenuList = append(sysAuthorityMenuList, model.SysAuthorityMenu{
//				MenuId:      v,
//				AuthorityId: strconv.FormatInt(in.AuthorityId, 10),
//			})
//		}
//
//		// 批量插入数据
//		if err := tx.Create(&sysAuthorityMenuList).Error; err != nil {
//			tx.Rollback()
//			return nil, err
//		}
//	}
//
//	// 提交事务
//	tx.Commit()
//
//	return &pb.AddAuthorityMenuResponse{}, nil
//}

//func (l *AddAuthorityMenuLogic) SetMenuAuthority(sysAuthorityauth *model.SysAuthority) error {
//	var s model.SysAuthority
//	l.svcCtx.DB.Preload("SysBaseMenus").First(&s, "authority_id = ?", sysAuthorityauth.AuthorityId)
//	err := l.svcCtx.DB.Model(&s).Association("SysBaseMenus").Replace(&sysAuthorityauth.SysBaseMenus)
//	return err
//}
