package model

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ CasbinRuleModel = (*customCasbinRuleModel)(nil)

type (
	// CasbinRuleModel is an interface to be customized, add more methods here,
	// and implement the added methods in customCasbinRuleModel.
	CasbinRuleModel interface {
		casbinRuleModel
		ListActivePolicies(ctx context.Context) ([]PolicyRow, error)
	}

	customCasbinRuleModel struct {
		*defaultCasbinRuleModel
	}
)

type PolicyRow struct {
	Ptype string `db:"ptype"`
	V0    string `db:"v0"`
	V1    string `db:"v1"`
	V2    string `db:"v2"`
	V3    string `db:"v3"`
	V4    string `db:"v4"`
	V5    string `db:"v5"`
}

// 保留直接授予 user:N 的 p 规则，同时排除已失效的角色、用户与部门。
func (m *customCasbinRuleModel) ListActivePolicies(ctx context.Context) ([]PolicyRow, error) {
	rows := make([]PolicyRow, 0)
	query := `SELECT c.ptype,c.v0,c.v1,c.v2,c.v3,c.v4,c.v5 FROM casbin_rule c
WHERE (c.ptype='p'
 AND (EXISTS (SELECT 1 FROM sys_role r WHERE r.code=c.v0 AND r.status=1 AND r.deleted_at IS NULL)
   OR EXISTS (SELECT 1 FROM sys_user u WHERE c.v0=CONCAT('user:',u.id) AND u.status=1 AND u.deleted_at IS NULL))
 AND (c.v1 IN ('*','dept:0') OR EXISTS (SELECT 1 FROM sys_dept d WHERE c.v1=CONCAT('dept:',d.id) AND d.status=1 AND d.deleted_at IS NULL)))
OR (c.ptype='g'
 AND EXISTS (SELECT 1 FROM sys_role r WHERE r.code=c.v1 AND r.status=1 AND r.deleted_at IS NULL)
 AND EXISTS (SELECT 1 FROM sys_user u WHERE c.v0=CONCAT('user:',u.id) AND u.status=1 AND u.deleted_at IS NULL)
 AND (c.v2 IN ('*','dept:0') OR EXISTS (SELECT 1 FROM sys_dept d WHERE c.v2=CONCAT('dept:',d.id) AND d.status=1 AND d.deleted_at IS NULL)))
ORDER BY c.id`
	err := m.QueryRowsNoCacheCtx(ctx, &rows, query)
	return rows, err
}

// NewCasbinRuleModel returns a model for the database table.
func NewCasbinRuleModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) CasbinRuleModel {
	return &customCasbinRuleModel{
		defaultCasbinRuleModel: newCasbinRuleModel(conn, c, opts...),
	}
}
