package userlogic

import (
	"context"
	"fmt"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"

	"github.com/gofrs/uuid/v5"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 新增（注册）用户 - 管理员
func (l *RegisterLogic) Register(in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	fmt.Println("--------- in", in)
	fmt.Println("--------- in", in.AuthorityIds)
	var authorities []model.SysAuthority
	for _, v := range in.AuthorityIds {
		authorities = append(authorities, model.SysAuthority{AuthorityId: v})
	}

	u := &model.SysUser{
		Username:    in.UserInfo.Username,
		NickName:    in.UserInfo.NickName,
		Password:    in.UserInfo.Password,
		HeaderImg:   in.UserInfo.HeaderImg,
		AuthorityId: in.UserInfo.AuthorityId,

		Authorities: authorities,
		Enable:      in.UserInfo.Enable,
		Phone:       in.UserInfo.Phone,
		Email:       in.UserInfo.Email,
	}

	err := l.svcCtx.DB.Where("username = ?", u.Username).First(&model.SysUser{}).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("用户名已注册")
	}

	// 否则 附加uuid 密码hash加密 注册
	u.Password = hash.BcryptHash(u.Password)
	u.UUID = uuid.Must(uuid.NewV4())
	err = l.svcCtx.DB.Create(&u).Error

	//var modelSysUser model.SysUser
	//_ = copier.Copy(&modelSysUser, in.UserInfo)
	// 附加uuid 密码hash加密 注册
	//modelSysUser.Password = hash.BcryptHash(modelSysUser.Password)
	//modelSysUser.UUID = uuid.Must(uuid.NewV4())
	//err = l.svcCtx.DB.Omit("deleted_at").Create(&modelSysUser).Error
	//err = l.svcCtx.DB.Create(&modelSysUser).Error

	var pbUserInfo pb.UserInfo
	_ = copier.Copy(&pbUserInfo, u)

	return &pb.RegisterResponse{
		UserInfo: &pbUserInfo,
	}, nil
}
