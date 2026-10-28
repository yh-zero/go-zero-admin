package authoritylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"strconv"
	"strings"
)

type GetAuthorityListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorityListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityListLogic {
	return &GetAuthorityListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuthorityListLogic) GetAuthorityList(in *pb.GetAuthorityListRequest) (*pb.GetAuthorityListResponse, error) {
	if in.Page == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "分页参数不能为空")
	}
	offset, size, err := accessutil.Page(in.Page.PageNo, in.Page.PageSize)
	if err != nil {
		return nil, err
	}
	var all []model.SysAuthority
	if err := l.svcCtx.DB.WithContext(l.ctx).Where("deleted_at IS NULL").Order("authority_id").Find(&all).Error; err != nil {
		return nil, err
	}
	var grants []model.SysAuthorityMenu
	if err := l.svcCtx.DB.WithContext(l.ctx).Order("sys_base_menu_id").Find(&grants).Error; err != nil {
		return nil, err
	}
	menus := map[string][]string{}
	for _, grant := range grants {
		menus[grant.AuthorityId] = append(menus[grant.AuthorityId], grant.MenuId)
	}
	roots := authorityTree(all, 0)
	result := &pb.GetAuthorityListResponse{Total: int64(len(roots)), SysAuthority: make([]*pb.SysAuthority, 0)}
	if offset >= len(roots) {
		return result, nil
	}
	end := offset + size
	if end > len(roots) {
		end = len(roots)
	}
	if err := copier.Copy(&result.SysAuthority, roots[offset:end]); err != nil {
		return nil, err
	}
	var fill func([]*pb.SysAuthority)
	fill = func(roles []*pb.SysAuthority) {
		for _, role := range roles {
			role.ShowMenuIds = strings.Join(menus[strconv.FormatInt(role.AuthorityId, 10)], ",")
			fill(role.Children)
		}
	}
	fill(result.SysAuthority)
	return result, nil
}

func authorityTree(all []model.SysAuthority, parent int64) []model.SysAuthority {
	children := make(map[int64][]int, len(all))
	for i := range all {
		p := int64(0)
		if all[i].ParentId != nil {
			p = *all[i].ParentId
		}
		children[p] = append(children[p], i)
	}
	var build func(int64) []model.SysAuthority
	build = func(parentID int64) []model.SysAuthority {
		indices := children[parentID]
		result := make([]model.SysAuthority, 0, len(indices))
		for _, i := range indices {
			role := all[i]
			role.Children = build(role.AuthorityId)
			result = append(result, role)
		}
		return result
	}
	return build(parent)
}
