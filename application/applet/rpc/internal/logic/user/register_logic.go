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
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/hash"
	"gorm.io/gorm"
	"strconv"
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
	if input == nil || strings.TrimSpace(input.Username) == "" {
		return nil, userError("用户名不能为空")
	}
	for _, field := range []struct{ name, value string }{
		{"username", strings.TrimSpace(input.Username)}, {"nickName", input.NickName},
		{"headerImg", input.HeaderImg}, {"phone", input.Phone}, {"email", input.Email},
	} {
		if err := validateProfileField(field.name, field.value); err != nil {
			return nil, err
		}
	}
	if err := hash.ValidatePassword(input.Password); err != nil {
		return nil, userError(err.Error())
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
	passwordHash, err := hash.BcryptHash(input.Password)
	if err != nil {
		return nil, err
	}
	user := model.SysUser{UUID: userUUID, Username: strings.TrimSpace(input.Username), Password: passwordHash, NickName: input.NickName, HeaderImg: input.HeaderImg, AuthorityId: input.AuthorityId, Enable: input.Enable, Phone: input.Phone, Email: input.Email}
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
		if err := replaceUserAuthorities(tx, user.ID, ids); err != nil {
			return err
		}
		return audit.Record(l.ctx, tx, audit.Event{Module: "user", Action: "register", Object: strconv.FormatInt(user.ID, 10)})
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
