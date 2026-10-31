package tenantservicelogic

import (
	"context"
	"encoding/hex"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

type ResetTenantAdminLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetTenantAdminLogic(ctx context.Context, s *svc.ServiceContext) *ResetTenantAdminLogic {
	return &ResetTenantAdminLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *ResetTenantAdminLogic) ResetTenantAdmin(in *tenantpb.ResetTenantAdminRequest) (*tenantpb.ResetTenantAdminResponse, error) {
	tenant, release, err := managedTenant(l.ctx, l.svcCtx, in.Actor, in.TenantID)
	if err != nil {
		return nil, err
	}
	defer release()
	if tenant.Status != "active" || tenant.AdminAccountReady != 1 {
		return nil, xerr.NewErrCodeMsg(400, "请先启动已初始化的租户")
	}
	ctx, cancel := context.WithTimeout(l.ctx, 90*time.Second)
	defer cancel()
	output, err := tenantSQL(ctx, l.svcCtx, tenant, strings.NewReader("SELECT HEX(username) FROM gozero_admin.sys_user WHERE id=1 AND deleted_at IS NULL;"))
	username, decodeErr := hex.DecodeString(strings.TrimSpace(string(output)))
	if err != nil || decodeErr != nil || len(username) == 0 {
		return nil, xerr.NewErrCodeMsg(500, "读取租户管理员失败")
	}
	random, err := jwtx.RandomToken()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	password := random[:24]
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	query := "UPDATE gozero_admin.sys_user SET password_hash=CONVERT(0x" + hex.EncodeToString(hash) + " USING utf8mb4),updated_at=NOW() WHERE id=1 AND deleted_at IS NULL; SELECT ROW_COUNT();"
	output, err = tenantSQL(ctx, l.svcCtx, tenant, strings.NewReader(query))
	if err != nil || strings.TrimSpace(string(output)) != "1" {
		return nil, xerr.NewErrCodeMsg(500, "更新租户管理员失败")
	}
	// 仅清理该租户 Redis 中管理员 ID/用户名缓存和会话，禁止全库清空。
	script := "local index='auth:{admin}:user:1';local ids=redis.call('ZRANGE',index,0,-1);for _,id in ipairs(ids) do redis.call('DEL','auth:{admin}:session:'..id) end;redis.call('DEL',index,'cache:gozeroAdminNew:sysUser:id:1','cache:gozeroAdminNew:sysUser:username:'..ARGV[1],'admin:login:acct:'..ARGV[1]);return 1"
	output, err = compose(ctx, l.svcCtx, tenant, nil, "exec", "-T", "redis", "redis-cli", "--raw", "EVAL", script, "0", string(username))
	if err != nil || strings.TrimSpace(string(output)) != "1" {
		return nil, xerr.NewErrCodeMsg(500, "管理员密码已更新，但清理旧会话失败，请重试重置")
	}
	return &tenantpb.ResetTenantAdminResponse{NewPassword: password}, nil
}
