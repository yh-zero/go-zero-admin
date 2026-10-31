package model

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

var _ AiWorkflowModel = (*customAiWorkflowModel)(nil)

type (
	// AiWorkflowModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAiWorkflowModel.
	AiWorkflowModel interface {
		aiWorkflowModel
		FindOwned(ctx context.Context, id, ownerId int64) (*WorkflowRow, error)
		ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*WorkflowRow, int64, error)
		UpdateOwned(ctx context.Context, id, ownerId int64, patch WorkflowPatch) error
		DeleteOwned(ctx context.Context, id, ownerId int64) error
	}

	customAiWorkflowModel struct {
		*defaultAiWorkflowModel
		conn sqlx.SqlConn
	}
)

// NewAiWorkflowModel returns a model for the database table.
func NewAiWorkflowModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AiWorkflowModel {
	return &customAiWorkflowModel{
		defaultAiWorkflowModel: newAiWorkflowModel(conn, c, opts...),
		conn:                   conn,
	}
}

type WorkflowRow struct {
	Id              int64  `db:"id"`
	Name            string `db:"name"`
	Definition      string `db:"definition"`
	IntervalSeconds int64  `db:"interval_seconds"`
	Enabled         int64  `db:"enabled"`
	NextRunAt       int64  `db:"next_run_at"`
	CreatedBy       int64  `db:"created_by"`
	CreatedAt       int64  `db:"created_at"`
	UpdatedAt       int64  `db:"updated_at"`
}

type WorkflowPatch struct {
	Name, Definition         *string
	IntervalSeconds, Enabled *int64
}

const workflowSelect = "SELECT id,name,CAST(definition AS CHAR) AS definition,interval_seconds,enabled,COALESCE(UNIX_TIMESTAMP(next_run_at),0) AS next_run_at,created_by,UNIX_TIMESTAMP(created_at) AS created_at,UNIX_TIMESTAMP(updated_at) AS updated_at FROM ai_workflow "

func (m *customAiWorkflowModel) FindOwned(ctx context.Context, id, ownerId int64) (*WorkflowRow, error) {
	var row WorkflowRow
	err := m.QueryRowNoCacheCtx(ctx, &row, workflowSelect+"WHERE id=? AND created_by=? AND deleted_at IS NULL", id, ownerId)
	return &row, err
}

func (m *customAiWorkflowModel) ListOwned(ctx context.Context, ownerId, page, pageSize int64) ([]*WorkflowRow, int64, error) {
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM ai_workflow WHERE created_by=? AND deleted_at IS NULL", ownerId); err != nil {
		return nil, 0, err
	}
	rows := make([]*WorkflowRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows, workflowSelect+"WHERE created_by=? AND deleted_at IS NULL ORDER BY id DESC LIMIT ? OFFSET ?", ownerId, pageSize, (page-1)*pageSize)
	return rows, count.Total, err
}

func (m *customAiWorkflowModel) UpdateOwned(ctx context.Context, id, ownerId int64, patch WorkflowPatch) error {
	sets := make([]string, 0, 4)
	args := make([]any, 0, 6)
	if patch.Name != nil {
		sets = append(sets, "name=?")
		args = append(args, *patch.Name)
	}
	if patch.Definition != nil {
		sets = append(sets, "definition=?")
		args = append(args, *patch.Definition)
	}
	if patch.IntervalSeconds != nil {
		sets = append(sets, "interval_seconds=?", "next_run_at=NULL")
		args = append(args, *patch.IntervalSeconds)
	}
	if patch.Enabled != nil {
		sets = append(sets, "enabled=?")
		args = append(args, *patch.Enabled)
	}
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_workflow WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		if len(sets) == 0 {
			return nil
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_workflow SET "+strings.Join(sets, ",")+" WHERE id=? AND created_by=?", append(args, id, ownerId)...)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowIdPrefix, id))
}

func (m *customAiWorkflowModel) DeleteOwned(ctx context.Context, id, ownerId int64) error {
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var owner struct {
			Id int64 `db:"id"`
		}
		if err := session.QueryRowCtx(ctx, &owner, "SELECT id FROM ai_workflow WHERE id=? AND created_by=? AND deleted_at IS NULL FOR UPDATE", id, ownerId); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE ai_workflow SET deleted_at=NOW(),enabled=0,next_run_at=NULL WHERE id=?", id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAiWorkflowIdPrefix, id))
}
