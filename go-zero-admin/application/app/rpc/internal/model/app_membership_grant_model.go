package model

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"strings"
)

var _ AppMembershipGrantModel = (*customAppMembershipGrantModel)(nil)

type AppMembershipGrantModel interface {
	appMembershipGrantModel
	List(context.Context, int64, string, bool, int64, int64) ([]MembershipRow, int64, error)
	Revoke(context.Context, uint64) error
}
type customAppMembershipGrantModel struct {
	*defaultAppMembershipGrantModel
}
type MembershipRow struct {
	Id        int64         `db:"id"`
	UserId    int64         `db:"user_id"`
	PlanCode  string        `db:"plan_code"`
	Benefits  string        `db:"benefits"`
	RequestId string        `db:"request_id"`
	GrantedBy sql.NullInt64 `db:"granted_by"`
	StartsAt  int64         `db:"starts_at"`
	ExpiresAt int64         `db:"expires_at"`
	Revoked   int64         `db:"revoked"`
	RevokedAt int64         `db:"revoked_at"`
	CreatedAt int64         `db:"created_at"`
}

func NewAppMembershipGrantModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AppMembershipGrantModel {
	return &customAppMembershipGrantModel{newAppMembershipGrantModel(conn, c, opts...)}
}
func (m *customAppMembershipGrantModel) List(ctx context.Context, userId int64, plan string, active bool, page, size int64) ([]MembershipRow, int64, error) {
	filters := []string{"1=1"}
	args := []any{}
	if userId > 0 {
		filters = append(filters, "user_id=?")
		args = append(args, userId)
	}
	if plan != "" {
		filters = append(filters, "plan_code=?")
		args = append(args, plan)
	}
	if active {
		filters = append(filters, "revoked=0 AND starts_at<=NOW() AND expires_at>NOW()")
	}
	where := strings.Join(filters, " AND ")
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM app_membership_grant WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := []MembershipRow{}
	query := "SELECT id,user_id,plan_code,benefits,request_id,granted_by,UNIX_TIMESTAMP(starts_at) AS starts_at,UNIX_TIMESTAMP(expires_at) AS expires_at,revoked,COALESCE(UNIX_TIMESTAMP(revoked_at),0) AS revoked_at,UNIX_TIMESTAMP(created_at) AS created_at FROM app_membership_grant WHERE " + where + " ORDER BY id DESC"
	if size > 0 {
		query += " LIMIT ? OFFSET ?"
		args = append(args, size, (page-1)*size)
	}
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, args...); err != nil {
		return nil, 0, err
	}
	return rows, count.Total, nil
}
func (m *customAppMembershipGrantModel) Revoke(ctx context.Context, id uint64) error {
	row, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE app_membership_grant SET revoked=1,revoked_at=COALESCE(revoked_at,NOW()),updated_at=NOW() WHERE id=?", id)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminAppMembershipGrantIdPrefix, id), cacheGozeroAdminAppMembershipGrantRequestIdPrefix+row.RequestId)
	return err
}
