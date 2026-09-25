package menulogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"strconv"
)

type GetMenuAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuAuthorityLogic {
	return &GetMenuAuthorityLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuAuthorityLogic) GetMenuAuthority(in *pb.GetMenuAuthorityRequest) (*pb.GetMenuAuthorityResponse, error) {
	if err := accessutil.RequireRole(l.svcCtx.DB.DB, in.AuthorityId); err != nil {
		return nil, err
	}
	var ids []string
	if err := l.svcCtx.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", in.AuthorityId).Pluck("sys_base_menu_id", &ids).Error; err != nil {
		return nil, err
	}
	var records []model.SysBaseMenu
	if len(ids) > 0 {
		if err := l.svcCtx.DB.Where("id IN ?", ids).Order("sort,id").Preload("Parameters").Preload("MenuBtn").Find(&records).Error; err != nil {
			return nil, err
		}
	}
	result := &pb.GetMenuAuthorityResponse{SysMenuList: make([]*pb.SysMenu, 0, len(records))}
	for _, record := range records {
		item := &pb.SysMenu{SysBaseMenu: &pb.SysBaseMenu{}, ID: record.ID}
		if err := copier.Copy(item.SysBaseMenu, record); err != nil {
			return nil, err
		}
		item.MenuId = strconv.FormatInt(record.ID, 10)
		item.AuthorityId = in.AuthorityId
		result.SysMenuList = append(result.SysMenuList, item)
	}
	return result, nil
}
