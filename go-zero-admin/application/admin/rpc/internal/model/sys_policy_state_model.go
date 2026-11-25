package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysPolicyStateModel = (*customSysPolicyStateModel)(nil)

type SysPolicyStateModel interface {
	CurrentRevision(ctx context.Context) (int64, error)
	BumpRevision(ctx context.Context) (int64, error)
}

type customSysPolicyStateModel struct {
	conn sqlx.SqlConn
}

func NewSysPolicyStateModel(conn sqlx.SqlConn, _ cache.CacheConf, _ ...cache.Option) SysPolicyStateModel {
	return &customSysPolicyStateModel{conn: conn}
}

func (m *customSysPolicyStateModel) CurrentRevision(ctx context.Context) (int64, error) {
	var revision int64
	err := m.conn.QueryRowCtx(ctx, &revision, "SELECT revision FROM sys_policy_state WHERE id=1")
	return revision, err
}

func (m *customSysPolicyStateModel) BumpRevision(ctx context.Context) (int64, error) {
	if _, err := m.conn.ExecCtx(ctx, "UPDATE sys_policy_state SET revision=revision+1,updated_at=NOW() WHERE id=1"); err != nil {
		return 0, err
	}
	return m.CurrentRevision(ctx)
}
