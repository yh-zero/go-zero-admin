package userlogic

import (
	"context"
	"errors"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid/v5"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"strings"
)

type RegisterLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterLogic {
	return &RegisterLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *RegisterLogic) Register(in *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	input := in.GetUserInfo()
	if input == nil || strings.TrimSpace(input.Username) == "" || len(input.Password) < 6 || len(input.Password) > 72 {
		return nil, userError("用户名不能为空，密码长度应为6至72字节")
	}
	if input.Enable == 0 {
		input.Enable = 1
	}
	if input.Enable != 1 && input.Enable != 2 {
		return nil, userError("用户状态只能为1或2")
	}
	userUUID, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	user := model.SysUser{UUID: userUUID, Username: strings.TrimSpace(input.Username), Password: hash.BcryptHash(input.Password), NickName: input.NickName, HeaderImg: input.HeaderImg, AuthorityId: input.AuthorityId, Enable: input.Enable, Phone: input.Phone, Email: input.Email}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		ids, err := validateUserAuthorities(tx, in.AuthorityIds, input.AuthorityId)
		if err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.SysUser{}).Where("username = ?", user.Username).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return userError("用户名已注册")
		}
		if err := tx.Omit("Authority", "Authorities").Create(&user).Error; err != nil {
			return err
		}
		return replaceUserAuthorities(tx, user.ID, ids)
	})
	if err != nil {
		return nil, registrationError(err)
	}
	return &pb.RegisterResponse{}, nil
}

// A concurrent registration can pass the read check; the database is the final guard.
func registrationError(err error) error {
	var mysqlError *mysqldriver.MySQLError
	if errors.As(err, &mysqlError) && mysqlError.Number == 1062 && strings.Contains(mysqlError.Message, "uk_sys_users_active_username") {
		return userError("用户名已注册")
	}
	return err
}
