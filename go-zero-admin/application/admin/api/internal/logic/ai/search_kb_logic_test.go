package ai

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSearchKbRejectsInvalidQueryAtAPIBoundary(t *testing.T) {
	for _, query := range []string{"", " \t\n　", strings.Repeat("资", 667), string([]byte{0xff})} {
		_, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{Validator: svc.NewValidator()}).SearchKb(&types.AiSearchKbReq{Id: 2, Query: query})
		ce, ok := xerr.AsCodeError(err)
		if !ok || ce.GetErrCode() != 400 {
			t.Fatalf("query bytes=%d error=%v, want 400", len(query), err)
		}
	}
}

type directRecallClient struct {
	aiservice.AiService
	search func(context.Context, *aipb.SearchKbRequest) (*aipb.SearchKbResponse, error)
}

func (c *directRecallClient) SearchKb(ctx context.Context, req *aipb.SearchKbRequest, _ ...grpc.CallOption) (*aipb.SearchKbResponse, error) {
	return c.search(ctx, req)
}

func TestSearchKbMapsRPCAndEmptyArray(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(7))
	for _, empty := range []bool{false, true} {
		client := &directRecallClient{search: func(ctx context.Context, req *aipb.SearchKbRequest) (*aipb.SearchKbResponse, error) {
			deadline, ok := ctx.Deadline()
			if !ok || time.Until(deadline) > 10*time.Second {
				t.Fatal("API deadline missing")
			}
			if req.Actor.UserId != 7 || req.KbId != 2 || req.Query != "资料" {
				t.Fatal("RPC actor/query lost")
			}
			out := &aipb.SearchKbResponse{ElapsedMs: 12}
			if !empty {
				out.List = []*aipb.KbSearchHit{{ChunkId: 31, KbId: 2, DocId: 20, DocTitle: "标题", Snippet: "片段", Score: .75}}
			}
			return out, nil
		}}
		out, err := NewSearchKbLogic(ctx, &svc.ServiceContext{Validator: svc.NewValidator(), AiRpc: client}).SearchKb(&types.AiSearchKbReq{Id: 2, Query: "资料"})
		if err != nil || out.List == nil || out.ElapsedMs != 12 {
			t.Fatalf("response=%+v err=%v", out, err)
		}
		if empty {
			if len(out.List) != 0 {
				t.Fatal("empty RPC result changed")
			}
		} else if len(out.List) != 1 || out.List[0].ChunkId != 31 || out.List[0].Snippet != "片段" || out.List[0].KbId != 2 || out.List[0].DocId != 20 || out.List[0].DocTitle != "标题" || out.List[0].Score != .75 {
			t.Fatal("hit mapping lost")
		}
	}
}

func TestSearchKbMissingActorAndTransportFailure(t *testing.T) {
	client := &directRecallClient{search: func(context.Context, *aipb.SearchKbRequest) (*aipb.SearchKbResponse, error) {
		t.Fatal("missing actor called RPC")
		return nil, nil
	}}
	_, err := NewSearchKbLogic(context.Background(), &svc.ServiceContext{Validator: svc.NewValidator(), AiRpc: client}).SearchKb(&types.AiSearchKbReq{Id: 2, Query: "资料"})
	ce, ok := xerr.AsCodeError(err)
	if !ok || ce.GetErrCode() != 403 {
		t.Fatalf("missing actor error=%v", err)
	}
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(7))
	for _, failure := range []error{status.Error(codes.Unavailable, "private RPC address"), status.Error(codes.DeadlineExceeded, "private timeout"), errors.New("private failure")} {
		client.search = func(context.Context, *aipb.SearchKbRequest) (*aipb.SearchKbResponse, error) { return nil, failure }
		_, err = NewSearchKbLogic(ctx, &svc.ServiceContext{Validator: svc.NewValidator(), AiRpc: client}).SearchKb(&types.AiSearchKbReq{Id: 2, Query: "资料"})
		ce, ok = xerr.AsCodeError(err)
		if !ok || ce.GetErrCode() != 503 || strings.Contains(err.Error(), "private") {
			t.Fatalf("unsafe transport error=%v", err)
		}
	}
}
