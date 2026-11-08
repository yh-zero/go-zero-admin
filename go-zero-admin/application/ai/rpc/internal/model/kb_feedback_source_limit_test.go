package model

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type sourceLimitSQL struct {
	sqlx.SqlConn
	calls int
	query string
	args  []any
}

func (s *sourceLimitSQL) QueryRowsCtx(_ context.Context, v any, query string, args ...any) error {
	s.calls++
	s.query = query
	s.args = append([]any{}, args...)
	rows := v.(*[]*FeedbackSourceRow)
	for _, id := range args[2:] {
		*rows = append(*rows, &FeedbackSourceRow{ChunkId: id.(int64), KbId: 5})
	}
	return nil
}
func TestFeedbackSourceLookupAcceptsEightAndRejectsNineBeforeSQL(t *testing.T) {
	conn := &sourceLimitSQL{}
	m := &customAiKbFeedbackModel{conn: conn, defaultAiKbFeedbackModel: &defaultAiKbFeedbackModel{CachedConn: sqlc.NewConnWithCache(conn, nil)}}
	ids := []int64{1, 2, 3, 4, 5, 6, 7, 8}
	rows, err := m.FindReadySourcesOwned(context.Background(), 7, 5, ids)
	if err != nil || len(rows) != 8 || conn.calls != 1 {
		t.Fatalf("八来源仍应一次SQL核验: rows=%d calls=%d err=%v", len(rows), conn.calls, err)
	}
	want := []any{int64(7), int64(5), int64(1), int64(2), int64(3), int64(4), int64(5), int64(6), int64(7), int64(8)}
	if !reflect.DeepEqual(conn.args, want) {
		t.Fatal("owner/kb/八个精确ID参数发生变化")
	}
	for _, clause := range []string{"k.owner_id=?", "c.kb_id=?", "d.kb_id=c.kb_id", "d.status='ready'", "d.deleted_at IS NULL", "k.deleted_at IS NULL", "c.status='ready'", "c.id IN (?,?,?,?,?,?,?,?)"} {
		if !strings.Contains(conn.query, clause) {
			t.Fatal("来源归属或状态过滤丢失")
		}
	}
	if _, err = m.FindReadySourcesOwned(context.Background(), 7, 5, append(ids, 9)); err != ErrFeedbackConflict || conn.calls != 1 {
		t.Fatal("九来源必须在SQL前拒绝")
	}
}
