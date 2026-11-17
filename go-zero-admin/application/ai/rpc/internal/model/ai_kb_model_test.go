package model

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type kbOwnershipSQL struct {
	sqlx.SqlConn
	calls int
	query string
	args  []any
	err   error
}

func (s *kbOwnershipSQL) QueryRowCtx(_ context.Context, target any, query string, args ...any) error {
	s.calls++
	s.query = query
	s.args = append([]any(nil), args...)
	if s.err != nil {
		return s.err
	}
	*target.(*int64) = 5
	return nil
}

func TestRequireOwnedAvoidsStatisticsAndPreservesFilters(t *testing.T) {
	for _, outcome := range []error{nil, ErrNotFound, errors.New("storage unavailable")} {
		conn := &kbOwnershipSQL{err: outcome}
		m := &customAiKbModel{conn: conn, defaultAiKbModel: &defaultAiKbModel{CachedConn: sqlc.NewConnWithCache(conn, nil)}}
		err := m.RequireOwned(context.Background(), 5, 7)
		if !errors.Is(err, outcome) || conn.calls != 1 {
			t.Fatalf("ownership result=%v calls=%d", err, conn.calls)
		}
		if !reflect.DeepEqual(conn.args, []any{int64(5), int64(7)}) {
			t.Fatal("ownership IDs must stay parameterized")
		}
		if conn.query != "SELECT id FROM ai_kb WHERE id=? AND owner_id=? AND deleted_at IS NULL" || strings.Contains(conn.query, "COUNT(") {
			t.Fatal("authorization must check the current owner and soft delete without document/chunk statistics")
		}
	}
}
