package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TenantRegistryModel = (*customTenantRegistryModel)(nil)

type (
	// TenantRegistryModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTenantRegistryModel.
	TenantRegistryModel interface {
		tenantRegistryModel
		List(ctx context.Context, page, pageSize int64, status string) ([]TenantListRow, int64, error)
		SetState(ctx context.Context, tenant *TenantRegistry, status, reason string, ready bool) error
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
	_, err := m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE tenant_registry SET status=?,error_msg=?,admin_account_ready=?,updated_at=NOW() WHERE id=? AND deleted_at IS NULL", status, reason, ready, tenant.Id)
	}, fmt.Sprintf("%s%d", cacheGozeroPlatformTenantRegistryIdPrefix, tenant.Id), cacheGozeroPlatformTenantRegistrySlugPrefix+tenant.Slug, cacheGozeroPlatformTenantRegistryDomainPrefix+tenant.Domain)
	return err
}

// NewTenantRegistryModel returns a model for the database table.
func NewTenantRegistryModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) TenantRegistryModel {
	return &customTenantRegistryModel{
		defaultTenantRegistryModel: newTenantRegistryModel(conn, c, opts...),
	}
}
