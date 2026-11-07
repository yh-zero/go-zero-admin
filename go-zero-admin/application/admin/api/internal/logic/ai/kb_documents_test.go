package ai

import (
	"context"
	"strings"
	"testing"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/client/aiservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
)

type documentClient struct {
	aiservice.AiService
	get    func(*aipb.GetKbDocRequest) (*aipb.GetKbDocResponse, error)
	update func(*aipb.UpdateKbDocRequest) (*aipb.UpdateKbDocResponse, error)
}

func (c *documentClient) GetKbDoc(_ context.Context, in *aipb.GetKbDocRequest, _ ...grpc.CallOption) (*aipb.GetKbDocResponse, error) {
	return c.get(in)
}
func (c *documentClient) UpdateKbDoc(_ context.Context, in *aipb.UpdateKbDocRequest, _ ...grpc.CallOption) (*aipb.UpdateKbDocResponse, error) {
	return c.update(in)
}

func TestKbDocumentsAPIMapsActorVersionStageAndMetadata(t *testing.T) {
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(9))
	doc := &aipb.KbDocInfo{Id: 3, KbId: 2, Title: "文档", Version: 4, Status: "pending", Stage: "indexing"}
	client := &documentClient{get: func(in *aipb.GetKbDocRequest) (*aipb.GetKbDocResponse, error) {
		if in.Actor.UserId != 9 || in.KbId != 2 || in.DocId != 3 || !in.MetaOnly {
			t.Fatal("GET lost actor, identity or view")
		}
		return &aipb.GetKbDocResponse{Doc: doc}, nil
	}, update: func(in *aipb.UpdateKbDocRequest) (*aipb.UpdateKbDocResponse, error) {
		if in.Actor.UserId != 9 || in.KbId != 2 || in.DocId != 3 || in.ExpectedVersion != 3 || in.Content != "原文\r\n" {
			t.Fatal("PUT lost actor, version or body")
		}
		return &aipb.UpdateKbDocResponse{Doc: doc}, nil
	}}
	s := &svc.ServiceContext{Validator: svc.NewValidator(), AiRpc: client}
	out, err := NewGetKbDocLogic(ctx, s).GetKbDoc(&types.AiGetKbDocReq{Id: 2, DocId: 3, MetaOnly: true})
	if err != nil || out.Doc.Version != 4 || out.Doc.Stage != "indexing" || out.Content != "" {
		t.Fatalf("GET=%+v err=%v", out, err)
	}
	updated, err := NewUpdateKbDocLogic(ctx, s).UpdateKbDoc(&types.AiUpdateKbDocReq{Id: 2, DocId: 3, Title: "文档", Content: "原文\r\n", ExpectedVersion: 3, RequestId: "f7cb82a2-3549-4680-b9c5-8fd295dd9e80"})
	if err != nil || updated.Doc.Stage != "indexing" || updated.Doc.Version != 4 {
		t.Fatalf("PUT=%+v err=%v", updated, err)
	}
}

func TestKbDocumentsAPIRejectsMissingFieldsAndOversizeUTF8(t *testing.T) {
	for _, req := range []*types.AiUpdateKbDocReq{
		{Id: 2, DocId: 3, Content: "正文", ExpectedVersion: 1}, {Id: 2, DocId: 3, Title: "文档", ExpectedVersion: 1},
		{Id: 2, DocId: 3, Title: "文档", Content: "正文"}, {Id: 2, DocId: 3, Title: "文档", Content: strings.Repeat("中", 21846), ExpectedVersion: 1},
	} {
		_, err := NewUpdateKbDocLogic(context.Background(), &svc.ServiceContext{Validator: svc.NewValidator()}).UpdateKbDoc(req)
		if ce, ok := xerr.AsCodeError(err); !ok || ce.GetErrCode() != 400 {
			t.Fatalf("invalid fields reached actor/RPC: %v", err)
		}
	}
}
