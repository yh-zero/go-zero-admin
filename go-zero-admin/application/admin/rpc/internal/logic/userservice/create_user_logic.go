package userservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateUserLogic) CreateUser(in *adminpb.CreateUserRequest) (*adminpb.NoDataResponse, error) {
	if in.AvatarFileId != 0 {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "请先创建用户，再编辑头像")
	}
	if in.GetActor().GetUserId() <= 0 || strings.TrimSpace(in.Username) == "" || in.Password == "" {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err := l.svcCtx.SysUserModel.FindOneByUsername(l.ctx, in.Username)
	if err == nil {
		return nil, xerr.NewErrCode(xerr.USERNAME_EXISTS)
	}
	if !errors.Is(err, sqlx.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	status := in.Status
	if status == 0 {
		status = 1
	}
	user := &model.SysUser{
		Username:     in.Username,
		PasswordHash: string(hash),
		Nickname:     in.Nickname,
		Gender:       int64(in.Gender),
		Status:       int64(status),
	}
	user.RealName = in.RealName
	user.Email = in.Email
	user.Phone = in.Phone
	if in.DeptId > 0 {
		user.DeptId = sql.NullInt64{Int64: in.DeptId, Valid: true}
	}
	if in.AvatarFileId > 0 {
		user.AvatarFileId = sql.NullInt64{Int64: in.AvatarFileId, Valid: true}
	}
	user.Remark = in.Remark

	if in.DeptId < 0 || in.AvatarFileId < 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	err = l.svcCtx.SysUserModel.CreateWithRoles(l.ctx, user, model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()}, in.RoleIds, nil)
	if err != nil {
		l.Error("创建用户失败")
		return nil, userMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}

func userMutationError(err error) error {
	if business, ok := xerr.AsCodeError(err); ok {
		return business
	}
	if errors.Is(err, model.ErrInvalidMutation) || errors.Is(err, model.ErrLastSuperAdmin) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
	}
	if errors.Is(err, model.ErrRoleAssignmentForbidden) || errors.Is(err, model.ErrDataScopeDenied) {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if errors.Is(err, sqlx.ErrNotFound) {
		return xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if errors.Is(err, model.ErrUsernameExists) {
		return xerr.NewErrCode(xerr.USERNAME_EXISTS)
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
}
