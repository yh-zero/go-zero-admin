package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"time"
)

var _ AppUserModel = (*customAppUserModel)(nil)

type (
	// AppUserModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAppUserModel.
	AppUserModel interface {
		appUserModel
		List(context.Context, string, string, int64, int64) ([]AppUserRow, int64, error)
		WithLoginUser(context.Context, uint64, string, func(context.Context) error) error
		SetNickname(context.Context, uint64, string) error
		SetReview(context.Context, uint64, string, string, func(context.Context) error) error
		SetAvatar(context.Context, uint64, int64, func(context.Context, int64, int64) error) error
		ReconcileAvatar(context.Context, uint64, func(context.Context, int64) error) error
	}

	customAppUserModel struct {
		*defaultAppUserModel
	}
)

// NewAppUserModel returns a model for the database table.
func NewAppUserModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) AppUserModel {
	return &customAppUserModel{
		defaultAppUserModel: newAppUserModel(conn, c, opts...),
	}
}

func (m *customAppUserModel) updateFields(ctx context.Context, id uint64, assignments string, args ...any) error {
	user, err := m.FindOne(ctx, id)
	if err != nil {
		return err
	}
	if user.DeletedAt.Valid {
		return ErrNotFound
	}
	_, err = m.ExecCtx(ctx, func(ctx context.Context, conn sqlx.SqlConn) (sql.Result, error) {
		return conn.ExecCtx(ctx, "UPDATE app_user SET "+assignments+",updated_at=NOW() WHERE id=? AND deleted_at IS NULL", append(args, id)...)
	}, fmt.Sprintf("%s%d", cacheGozeroAdminAppUserIdPrefix, id), cacheGozeroAdminAppUserAccountPrefix+user.Account)
	return err
}

var (
	ErrAppLoginChanged = errors.New("app user credentials changed")
	ErrAppUserInactive = errors.New("app user is not active")
)

// WithLoginUser 把会话创建与审核串行化，缓存中的旧密码或状态不能签发新会话。
func (m *customAppUserModel) WithLoginUser(ctx context.Context, id uint64, expectedHash string, issue func(context.Context) error) error {
	var account string
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var row struct {
			Account      string `db:"account"`
			PasswordHash string `db:"password_hash"`
			Status       string `db:"status"`
		}
		if err := session.QueryRowCtx(ctx, &row, "SELECT account,password_hash,status FROM app_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", id); err != nil {
			return err
		}
		if row.PasswordHash != expectedHash {
			return ErrAppLoginChanged
		}
		if row.Status != "active" {
			return ErrAppUserInactive
		}
		account = row.Account
		if err := issue(ctx); err != nil {
			return err
		}
		_, err := session.ExecCtx(ctx, "UPDATE app_user SET last_login_at=?,updated_at=NOW() WHERE id=?", time.Now(), id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAppUserIdPrefix, id), cacheGozeroAdminAppUserAccountPrefix+account)
}
func (m *customAppUserModel) SetNickname(ctx context.Context, id uint64, name string) error {
	return m.updateFields(ctx, id, "nickname=?", name)
}
func (m *customAppUserModel) SetReview(ctx context.Context, id uint64, status, reason string, revoke func(context.Context) error) error {
	var account string
	err := m.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var row struct {
			Account string `db:"account"`
			Status  string `db:"status"`
		}
		if err := session.QueryRowCtx(ctx, &row, "SELECT account,status FROM app_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", id); err != nil {
			return err
		}
		account = row.Account
		// 撤销失败时回滚审核；与登录共用行锁，禁止在撤销之后补发旧状态会话。
		if row.Status != status || status != "active" {
			if err := revoke(ctx); err != nil {
				return err
			}
		}
		_, err := session.ExecCtx(ctx, "UPDATE app_user SET status=?,review_reason=?,updated_at=NOW() WHERE id=?", status, reason, id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAppUserIdPrefix, id), cacheGozeroAdminAppUserAccountPrefix+account)
}
func (m *customAppUserModel) SetAvatar(ctx context.Context, id uint64, fileId int64, prepare func(context.Context, int64, int64) error) error {
	var account string
	err := m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var row struct {
			Account      string `db:"account"`
			AvatarFileId int64  `db:"avatar_file_id"`
		}
		if err := s.QueryRowCtx(ctx, &row, "SELECT account,COALESCE(avatar_file_id,0) AS avatar_file_id FROM app_user WHERE id=? AND deleted_at IS NULL FOR UPDATE", id); err != nil {
			return err
		}
		account = row.Account
		if err := prepare(ctx, row.AvatarFileId, fileId); err != nil {
			return err
		}
		_, err := s.ExecCtx(ctx, "UPDATE app_user SET avatar_file_id=? WHERE id=? AND deleted_at IS NULL", sql.NullInt64{Int64: fileId, Valid: fileId > 0}, id)
		return err
	})
	if err != nil {
		return err
	}
	return m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminAppUserIdPrefix, id), cacheGozeroAdminAppUserAccountPrefix+account)
}

func (m *customAppUserModel) ReconcileAvatar(ctx context.Context, id uint64, reconcile func(context.Context, int64) error) error {
	return m.TransactCtx(ctx, func(ctx context.Context, s sqlx.Session) error {
		var row struct {
			AvatarFileId int64        `db:"avatar_file_id"`
			DeletedAt    sql.NullTime `db:"deleted_at"`
		}
		err := s.QueryRowCtx(ctx, &row, "SELECT COALESCE(avatar_file_id,0) AS avatar_file_id,deleted_at FROM app_user WHERE id=? FOR UPDATE", id)
		if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
			return err
		}
		keep := row.AvatarFileId
		if errors.Is(err, sqlx.ErrNotFound) || row.DeletedAt.Valid {
			keep = 0
		}
		return reconcile(ctx, keep)
	})
}

type AppUserRow struct {
	Id            int64  `db:"id"`
	Account       string `db:"account"`
	IdentityKind  string `db:"identity_kind"`
	Nickname      string `db:"nickname"`
	AvatarFileId  int64  `db:"avatar_file_id"`
	Status        string `db:"status"`
	ReviewReason  string `db:"review_reason"`
	EmailVerified bool   `db:"email_verified"`
	PhoneVerified bool   `db:"phone_verified"`
	LastLoginAt   int64  `db:"last_login_at"`
	CreatedAt     int64  `db:"created_at"`
}

func (m *customAppUserModel) List(ctx context.Context, status, account string, page, size int64) ([]AppUserRow, int64, error) {
	where := "deleted_at IS NULL"
	args := []any{}
	if status != "" {
		where += " AND status=?"
		args = append(args, status)
	}
	if account != "" {
		where += " AND account LIKE ?"
		args = append(args, "%"+account+"%")
	}
	var count struct {
		Total int64 `db:"total"`
	}
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM app_user WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	rows := []AppUserRow{}
	query := "SELECT id,account,identity_kind,nickname,COALESCE(avatar_file_id,0) AS avatar_file_id,status,review_reason,email_verified_at IS NOT NULL AS email_verified,phone_verified_at IS NOT NULL AS phone_verified,COALESCE(UNIX_TIMESTAMP(last_login_at),0) AS last_login_at,UNIX_TIMESTAMP(created_at) AS created_at FROM app_user WHERE " + where + " ORDER BY id DESC LIMIT ? OFFSET ?"
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, query, append(args, size, (page-1)*size)...); err != nil {
		return nil, 0, err
	}
	return rows, count.Total, nil
}
