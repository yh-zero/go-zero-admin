package authoritylogic

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strconv"
)

type CreateAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAuthorityLogic {
	return &CreateAuthorityLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateAuthorityLogic) CreateAuthority(in *pb.CreateAuthorityRequest) (*pb.CreateAuthorityResponse, error) {
	var role model.SysAuthority
	err := accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		if err := validateAuthority(tx, in.SysAuthority); err != nil {
			return err
		}
		if err := accessutil.Unique(tx, &model.SysAuthority{}, "authority_id = ?", in.SysAuthority.AuthorityId); err != nil {
			return err
		}
		home := in.SysAuthority.DefaultRouter
		if home == "" {
			home = "index"
		}
		var menu model.SysBaseMenu
		if err := tx.Where("name = ?", home).First(&menu).Error; err != nil {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "默认首页菜单不存在")
		}
		if menu.ParentId != 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "新角色默认首页请选择根级菜单，创建后可分配其他菜单再修改")
		}
		parent := in.SysAuthority.ParentId
		role = model.SysAuthority{AuthorityId: in.SysAuthority.AuthorityId, AuthorityName: in.SysAuthority.AuthorityName, ParentId: &parent, DefaultRouter: home}
		if err := tx.Omit("SysBaseMenus", "DataAuthorityId", "Users").Create(&role).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.SysAuthorityMenu{MenuId: strconv.FormatInt(menu.ID, 10), AuthorityId: strconv.FormatInt(role.AuthorityId, 10)}).Error; err != nil {
			return err
		}
		return tx.Create(&gormadapter.CasbinRule{Ptype: "p", V0: strconv.FormatInt(role.AuthorityId, 10), V1: "/v1/sys/menu/getMenu", V2: "GET"}).Error
	})
	if err != nil {
		return nil, err
	}
	result := &pb.SysAuthority{}
	if err := copier.Copy(result, role); err != nil {
		return nil, err
	}
	return &pb.CreateAuthorityResponse{SysAuthority: result}, nil
}
