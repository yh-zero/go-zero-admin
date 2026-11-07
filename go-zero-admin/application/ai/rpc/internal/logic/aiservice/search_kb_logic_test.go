package aiservicelogic

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
)

type searchOwnedModel struct {
	model.AiKbModel
	calls int
	err   error
}

func (m *searchOwnedModel) FindOwned(_ context.Context, kb, owner int64) (*model.KbRow, error) {
	m.calls++
	if kb != 2 || owner != 7 {
		return nil, model.ErrNotFound
	}
	return &model.KbRow{Id: kb, OwnerId: owner}, m.err
}

func assertSearchCode(t *testing.T, err error, code uint32) {
	t.Helper()
	ce, ok := xerr.AsCodeError(err)
	if !ok || ce.GetErrCode() != code {
		t.Fatalf("error=%v, want code=%d", err, code)
	}
}

func TestSearchKbRejectsInvalidBeforeDependencies(t *testing.T) {
	for _, tc := range []struct {
		name string
		req  *aipb.SearchKbRequest
		code uint32
	}{
		{"no_actor", &aipb.SearchKbRequest{KbId: 2, Query: "资料"}, 403},
		{"bad_id", &aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, Query: "资料"}, 400},
		{"blank", &aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 2, Query: " \t\n　"}, 400},
		{"utf8_bytes", &aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 2, Query: strings.Repeat("资", 667)}, 400},
		{"invalid_utf8", &aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 2, Query: string([]byte{0xff})}, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &searchOwnedModel{}
			_, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{AiKbModel: m}).SearchKb(tc.req)
			assertSearchCode(t, err, tc.code)
			if m.calls != 0 {
				t.Fatal("invalid request accessed model")
			}
		})
	}
}

func TestSearchKbChecksOwnershipBeforeDependencies(t *testing.T) {
	for _, kb := range []int64{2, 3} {
		m := &searchOwnedModel{}
		_, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{AiKbModel: m}).SearchKb(&aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: kb, Query: "资料"})
		code := uint32(503)
		if kb == 3 {
			code = 404
		}
		assertSearchCode(t, err, code)
		if m.calls != 1 {
			t.Fatal("ownership must be checked exactly once")
		}
	}
}

type directRecallSearcher struct {
	calls  int
	search func(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error)
}

func (s *directRecallSearcher) Search(ctx context.Context, owner, kb int64, query string) ([]agenttools.KbSearchHit, error) {
	s.calls++
	return s.search(ctx, owner, kb, query)
}

func TestSearchKbMapsHitsAndPropagatesBoundedContext(t *testing.T) {
	query := strings.Repeat("a", 2000)
	searcher := &directRecallSearcher{search: func(ctx context.Context, owner, kb int64, got string) ([]agenttools.KbSearchHit, error) {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 10*time.Second || time.Until(deadline) <= 0 {
			t.Fatal("missing bounded deadline")
		}
		if owner != 7 || kb != 2 || got != query {
			t.Fatal("search parameters lost")
		}
		return []agenttools.KbSearchHit{{ChunkID: 31, KbID: 2, DocID: 20, DocTitle: "标题", Snippet: "真实片段", Score: .75}}, nil
	}}
	out, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{AiKbModel: &searchOwnedModel{}, KbSearcher: searcher}).SearchKb(&aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 2, Query: query})
	if err != nil {
		t.Fatal(err)
	}
	if len(out.List) != 1 || out.List[0].ChunkId != 31 || out.List[0].KbId != 2 || out.List[0].DocId != 20 || out.List[0].DocTitle != "标题" || out.List[0].Snippet != "真实片段" || out.List[0].Score != .75 || out.ElapsedMs < 0 || searcher.calls != 1 {
		t.Fatalf("unexpected response %+v", out)
	}
}

func TestSearchKbDistinguishesEmptyFailureAndOwnership(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		kb                    int64
		modelErr, errorResult error
		code                  uint32
	}{
		{name: "empty", kb: 2},
		{name: "upstream_failure", kb: 2, errorResult: errors.New("private upstream detail"), code: 503},
		{name: "not_found", kb: 3, code: 404},
		{name: "other_owner", kb: 2, modelErr: model.ErrNotFound, code: 404},
		{name: "db_failure", kb: 2, modelErr: errors.New("private DB detail"), code: 505},
	} {
		t.Run(tc.name, func(t *testing.T) {
			searcher := &directRecallSearcher{search: func(context.Context, int64, int64, string) ([]agenttools.KbSearchHit, error) {
				return nil, tc.errorResult
			}}
			out, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{AiKbModel: &searchOwnedModel{err: tc.modelErr}, KbSearcher: searcher}).SearchKb(&aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: tc.kb, Query: "资料"})
			if tc.code == 0 {
				if err != nil || out.List == nil || len(out.List) != 0 {
					t.Fatalf("empty result=%+v err=%v", out, err)
				}
			} else {
				assertSearchCode(t, err, tc.code)
				if strings.Contains(err.Error(), "private") {
					t.Fatal("internal failure exposed")
				}
			}
			want := 1
			if tc.code == 404 || tc.code == 505 {
				want = 0
			}
			if searcher.calls != want {
				t.Fatal("ownership failure called paid dependency")
			}
		})
	}
}

func TestSearchKbHonorsEarlierCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Millisecond)
	defer cancel()
	searcher := &directRecallSearcher{search: func(ctx context.Context, _ int64, _ int64, _ string) ([]agenttools.KbSearchHit, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	}}
	_, err := NewSearchKbLogic(ctx, &svc.ServiceContext{AiKbModel: &searchOwnedModel{}, KbSearcher: searcher}).SearchKb(&aipb.SearchKbRequest{Actor: &aipb.Actor{UserId: 7}, KbId: 2, Query: "资料"})
	assertSearchCode(t, err, 503)
}
