package userlogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/pkg/result/xerr"
	"go-zero-admin/pkg/utils/hash"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取用户信息
func (l *GetUserInfoLogic) GetUserInfo(in *pb.GetUserInfoRequest) (*pb.GetUserInfoResponse, error) {
	var userInfo model.SysUser
	userInfoModel := &pb.UserInfo{}
	err := l.svcCtx.DB.Where("username = ?", in.UserName).Preload("Authorities").Preload("Authority").First(&userInfo).Error
	if err == nil {
		if ok := hash.BcryptCheck(in.Password, userInfo.Password); !ok {
			return nil, errors.Wrapf(xerr.NewErrCode(xerr.USER_PASSWORD_ERROR), "utils.BcryptCheck(in.Password, userInfo.Password)")
		}
		l.UserAuthorityDefaultRouter(&userInfo)
	}

	if err := copier.Copy(userInfoModel, &userInfo); err != nil {
		return nil, err
	}
	return &pb.GetUserInfoResponse{
		UserInfo: userInfoModel,
	}, err
}

// UserAuthorityDefaultRouter 用户角色默认路由检查
func (l *GetUserInfoLogic) UserAuthorityDefaultRouter(user *model.SysUser) {
	var menuIds []string
	err := l.svcCtx.DB.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", user.AuthorityId).Pluck("sys_base_menu_id", &menuIds).Error
	if err != nil {
		return
	}
	var sysBaseMenuModel model.SysBaseMenu
	err = l.svcCtx.DB.First(&sysBaseMenuModel, "name = ? and id in (?)", user.Authority.DefaultRouter, menuIds).Error // 查到sys_authorities的DefaultRouter  用他去查SysBaseMenu  然后判断有没有默认的路由

	if errors.Is(err, gorm.ErrRecordNotFound) {
		user.Authority.DefaultRouter = "404"
	}
}
