package authoritylogic

import (
	"context"
	"gorm.io/gorm"
	"strconv"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	modelBase "go-zero-admin/pkg/model"

	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAuthorityLogic {
	return &CreateAuthorityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建角色
func (l *CreateAuthorityLogic) CreateAuthority(in *pb.CreateAuthorityRequest) (*pb.CreateAuthorityResponse, error) {
	var auth model.SysAuthority
	_ = copier.Copy(&auth, in.SysAuthority)
	err := l.svcCtx.DB.Where("authority_id = ?", auth.AuthorityId).First(&model.SysAuthority{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("已存在 authority_id")
	}

	txErr := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if err = tx.Omit("deleted_at").Create(&auth).Error; err != nil {
			return err
		}

		auth.SysBaseMenus = l.DefaultMenu()
		if err = tx.Model(&auth).Association("SysBaseMenus").Replace(&auth.SysBaseMenus); err != nil {
			return err
		}
		casbinInfos := l.DefaultCasbin()
		authorityId := strconv.Itoa(int(auth.AuthorityId))
		rules := [][]string{}
		for _, v := range casbinInfos {
			rules = append(rules, []string{authorityId, v.Path, v.Method})
		}
		return l.AddPolicies(tx, rules)
	})

	var pbSysAuthority pb.SysAuthority
	_ = copier.Copy(&pbSysAuthority, auth)

	return &pb.CreateAuthorityResponse{SysAuthority: &pbSysAuthority}, txErr
}

func (l *CreateAuthorityLogic) DefaultMenu() []model.SysBaseMenu {
	return []model.SysBaseMenu{{
		MODEL_BASE: modelBase.MODEL_BASE{ID: 1},
		ParentId:   0,
		Path:       "routerHolder",
		Name:       "routerHolder",
		Component:  "views/routerHolder.vue",
		Sort:       1,
		Meta: model.Meta{
			Title: "默认页",
			Icon:  "none",
		},
	}}
}

func (l *CreateAuthorityLogic) DefaultCasbin() []*pb.CasbinInfo {
	return []*pb.CasbinInfo{
		{Path: "/v1/sys/login", Method: "POST"},        // 登录
		{Path: "/v1/sys/menu/getMenu", Method: "POST"}, // 获取页面菜单 - 路由

		{Path: "/user/admin_register", Method: "POST"},
		{Path: "/user/changePassword", Method: "POST"},
		{Path: "/user/setUserAuthority", Method: "POST"},
		{Path: "/user/setUserInfo", Method: "PUT"},
		{Path: "/user/getUserInfo", Method: "GET"},
		//{Path: "/jwt/jsonInBlacklist", Method: "POST"},
	}
}

// 添加匹配的权限
func (l *CreateAuthorityLogic) AddPolicies(db *gorm.DB, rules [][]string) error {
	var casbinRules []gormadapter.CasbinRule
	for i := range rules {
		casbinRules = append(casbinRules, gormadapter.CasbinRule{
			Ptype: "p",
			V0:    rules[i][0],
			V1:    rules[i][1],
			V2:    rules[i][2],
		})
	}
	return db.Create(&casbinRules).Error
}
