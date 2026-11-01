package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ AppFeedbackModel = (*customAppFeedbackModel)(nil)

type (
	// AppFeedbackModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAppFeedbackModel.
	AppFeedbackModel interface {
		appFeedbackModel
		ListPage(context.Context, int64, string, int64, int64) ([]*AppFeedbackRow, int64, error)
		Patch(context.Context, uint64, *string, *string) error
	}

	customAppFeedbackModel struct {
		*defaultAppFeedbackModel
	}
)

// NewAppFeedbackModel returns a model for the database table.
func NewAppFeedbackModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AppFeedbackModel {
	return &customAppFeedbackModel{
		defaultAppFeedbackModel: newAppFeedbackModel(conn, c, opts...),
	}
}

type AppFeedbackRow struct {
	Id        int64  `db:"id"`
	UserId    int64  `db:"user_id"`
	Account   string `db:"account"`
	Nickname  string `db:"nickname"`
	Content   string `db:"content"`
	Reply     string `db:"reply"`
	Status    string `db:"status"`
	CreatedAt int64  `db:"created_at"`
	UpdatedAt int64  `db:"updated_at"`
}

func (m *customAppFeedbackModel) ListPage(ctx context.Context, userId int64, status string, page, size int64) ([]*AppFeedbackRow, int64, error) {
	where := "1=1"
	var args []any
	if userId > 0 {
		where += " AND f.user_id=?"
		args = append(args, userId)
	}
	if status != "" {
		where += " AND f.status=?"
		args = append(args, status)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM app_feedback f WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := make([]*AppFeedbackRow, 0)
	query := "SELECT f.id,f.user_id,COALESCE(u.account,'') AS account,COALESCE(u.nickname,'') AS nickname,f.content,f.reply,f.status,UNIX_TIMESTAMP(f.created_at) AS created_at,UNIX_TIMESTAMP(f.updated_at) AS updated_at FROM app_feedback f LEFT JOIN app_user u ON u.id=f.user_id WHERE " + where + " ORDER BY f.id DESC LIMIT ? OFFSET ?"
	err := m.QueryRowsNoCacheCtx(ctx, &rows, query, append(args, size, (page-1)*size)...)
	return rows, total, err
}

func (m *customAppFeedbackModel) Patch(ctx context.Context, id uint64, reply, status *string) error {
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var currentId uint64
		if err := s.QueryRowCtx(ctx, &currentId, "SELECT id FROM app_feedback WHERE id=? FOR UPDATE", id); err != nil {
			return err
		}
		var assignments []string
		var args []any
		if reply != nil {
			assignments, args = append(assignments, "reply=?"), append(args, *reply)
		}
		if status != nil {
			assignments, args = append(assignments, "status=?"), append(args, *status)
		}
		if len(assignments) == 0 {
			return nil
		}
		_, err := s.ExecCtx(ctx, "UPDATE app_feedback SET "+strings.Join(assignments, ",")+" WHERE id=?", append(args, id)...)
		return err
	})
	if err != nil {
		return err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAppFeedbackIdPrefix, id)); err != nil {
		return fmt.Errorf("%w: %v", ErrCommittedCacheInvalidation, err)
	}
	return nil
}
