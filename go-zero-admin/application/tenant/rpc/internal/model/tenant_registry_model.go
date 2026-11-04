package model

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TenantRegistryModel = (*customTenantRegistryModel)(nil)

var ErrStepCacheInvalidation = errors.New("tenant step write succeeded but cache invalidation failed")

type (
	// TenantRegistryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTenantRegistryModel.
	TenantRegistryModel interface {
		tenantRegistryModel
		List(ctx context.Context, page, pageSize int64, status string) ([]TenantListRow, int64, error)
		SetState(ctx context.Context, tenant *TenantRegistry, status, reason string, ready bool) error
		SetStep(ctx context.Context, tenant *TenantRegistry, step string) error
	}

	customTenantRegistryModel struct {
		*defaultTenantRegistryModel
	}
)

type TenantListRow struct {
	Id                int64  `db:"id"`
	Slug              string `db:"slug"`
	Domain            string `db:"domain"`
	ComposeProject    string `db:"compose_project"`
	Status            string `db:"status"`
	ErrorMsg          string `db:"error_msg"`
	CreatedBy         int64  `db:"created_by"`
	CreatedAt         int64  `db:"created_at"`
	UpdatedAt         int64  `db:"updated_at"`
	AdminAccountReady int64  `db:"admin_account_ready"`
}

func (m *customTenantRegistryModel) List(ctx context.Context, page, size int64, status string) ([]TenantListRow, int64, error) {
	where := "deleted_at IS NULL"
	args := []any{}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM tenant_registry WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]TenantListRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT id,slug,domain,compose_project,status,error_msg,COALESCE(created_by,0) AS created_by,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at,admin_account_ready FROM tenant_registry WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?", append(args, size, (page-1)*size)...)
	return rows, count.Total, err
}

func (m *customTenantRegistryModel) SetState(ctx context.Context, tenant *TenantRegistry, status, reason string, ready bool) error {
	if utf8.RuneCountInString(reason) > 512 {
		reason = string([]rune(reason)[:512])
	}
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE tenant_registry SET status=?,error_msg=?,admin_account_ready=?,updated_at=NOW() WHERE id=? AND deleted_at IS NULL", status, reason, ready, tenant.Id)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminTenantRegistryIdPrefix, tenant.Id), cacheGozeroAdminTenantRegistrySlugPrefix+tenant.Slug, cacheGozeroAdminTenantRegistryDomainPrefix+tenant.Domain)
	return err
}

func (m *customTenantRegistryModel) SetStep(ctx context.Context, tenant *TenantRegistry, step string) error {
	result, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE tenant_registry SET current_step=?,step_updated_at=NOW() WHERE id=? AND deleted_at IS NULL", step, tenant.Id)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminTenantRegistryIdPrefix, tenant.Id), cacheGozeroAdminTenantRegistrySlugPrefix+tenant.Slug, cacheGozeroAdminTenantRegistryDomainPrefix+tenant.Domain)
	if err != nil && result != nil {
		return fmt.Errorf("%w: %w", ErrStepCacheInvalidation, err)
	}
	return err
}

// InitializeTenantDatabase 仅初始化尚未对外开放的独立租户库；执行器由部署逻辑提供，不使用平台库连接。
func InitializeTenantDatabase(ctx context.Context, seed io.Reader, username, passwordHash string, execute func(context.Context, io.Reader) ([]byte, error)) error {
	if _, err := execute(ctx, seed); err != nil {
		return err
	}
	cleanup := "USE gozero_admin; SET FOREIGN_KEY_CHECKS=0;\n"
	for _, table := range []string{"ai_message", "ai_task", "ai_conversation", "ai_workflow", "app_membership_grant", "app_feedback", "app_user", "sys_file_reference", "sys_file", "sys_audit_log"} {
		cleanup += "DELETE FROM " + table + ";\n"
	}
	cleanup += "DELETE FROM sys_user WHERE id<>1;\nDELETE FROM sys_dept WHERE id<>1;\n"
	cleanup += "DELETE FROM sys_role_menu WHERE role_id<>1;\n"
	cleanup += "DELETE FROM sys_menu WHERE perm_code LIKE 'sys:tenant:%' OR name IN ('tenant','tenant_list');\n"
	cleanup += "DELETE FROM sys_role_menu WHERE menu_id NOT IN (SELECT id FROM sys_menu);\nDELETE FROM sys_role WHERE id<>1;\n"
	cleanup += "DELETE FROM casbin_rule WHERE NOT ((ptype='g' AND v0='user:1' AND v1='super_admin' AND v2='*') OR (ptype='p' AND v0='super_admin'));\n"
	cleanup += "UPDATE sys_user SET username=CONVERT(0x" + hex.EncodeToString([]byte(username)) + " USING utf8mb4),password_hash=CONVERT(0x" + hex.EncodeToString([]byte(passwordHash)) + " USING utf8mb4),nickname='管理员',real_name='',email='',phone='',status=1,dept_id=1,avatar_file_id=NULL,last_login_at=NULL,last_login_ip='',remark='',deleted_at=NULL,updated_at=NOW() WHERE id=1;\n"
	cleanup += "SET FOREIGN_KEY_CHECKS=1;\nSELECT COUNT(*) FROM sys_user WHERE id=1 AND status=1 AND deleted_at IS NULL;\n"
	result, err := execute(ctx, strings.NewReader(cleanup))
	if err != nil || strings.TrimSpace(string(result)) != "1" {
		return errors.New("tenant initialization failed")
	}
	return nil
}

// ResetTenantAdminDatabase 仅操作已授权租户的管理员，跨容器执行器由逻辑层提供。
func ResetTenantAdminDatabase(ctx context.Context, passwordHash string, execute func(context.Context, io.Reader) ([]byte, error)) (string, error) {
	output, err := execute(ctx, strings.NewReader("SELECT HEX(username) FROM gozero_admin.sys_user WHERE id=1 AND deleted_at IS NULL;"))
	username, decodeErr := hex.DecodeString(strings.TrimSpace(string(output)))
	if err != nil || decodeErr != nil || len(username) == 0 {
		return "", errors.New("读取租户管理员失败")
	}
	// 执行器使用 mysql 标准输入；值以十六进制字面量编码，不能拼接原始值。
	query := "UPDATE gozero_admin.sys_user SET password_hash=CONVERT(0x" + hex.EncodeToString([]byte(passwordHash)) + " USING utf8mb4),updated_at=NOW() WHERE id=1 AND deleted_at IS NULL; SELECT ROW_COUNT();"
	output, err = execute(ctx, strings.NewReader(query))
	if err != nil || strings.TrimSpace(string(output)) != "1" {
		return "", errors.New("更新租户管理员失败")
	}
	return string(username), nil
}

// NewTenantRegistryModel returns a model for the database table.
func NewTenantRegistryModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TenantRegistryModel {
	return &customTenantRegistryModel{
		defaultTenantRegistryModel: newTenantRegistryModel(conn, c, opts...),
	}
}
