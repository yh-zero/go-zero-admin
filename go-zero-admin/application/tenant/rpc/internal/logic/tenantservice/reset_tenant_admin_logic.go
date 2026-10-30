package tenantservicelogic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type ResetTenantAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetTenantAdminLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetTenantAdminLogic {
	return &ResetTenantAdminLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ResetTenantAdminLogic) ResetTenantAdmin(in *tenantpb.ResetTenantAdminRequest) (*tenantpb.ResetTenantAdminResponse, error) {
	tenant, err := l.svcCtx.TenantRegistryModel.FindOne(l.ctx, uint64(in.TenantID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(404, "租户不存在")
	}
	// 连租户库重置 admin 密码
	dsn := fmt.Sprintf("root:%s@tcp(%s-api:3306)/gozero_admin", tenant.MysqlRootPassword, tenant.Slug)
	conn := sqlx.NewMysql(dsn)
	newPassword := generatePassword(16)
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	_, err = conn.ExecCtx(l.ctx, "UPDATE sys_user SET password_hash=?, updated_at=NOW() WHERE username='admin' AND deleted_at IS NULL", string(hash))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &tenantpb.ResetTenantAdminResponse{NewPassword: newPassword}, nil
}
