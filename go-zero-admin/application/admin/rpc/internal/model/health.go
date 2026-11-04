package model

import (
	"context"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func CheckDatabaseHealth(ctx context.Context, conn sqlx.SqlConn) error {
	var row struct {
		Value int64 `db:"value"`
	}
	return conn.QueryRowCtx(ctx, &row, "SELECT 1 AS value")
}
