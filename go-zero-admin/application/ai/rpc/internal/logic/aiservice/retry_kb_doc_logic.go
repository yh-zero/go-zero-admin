package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

type RetryKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRetryKbDocLogic(ctx context.Context, s *svc.ServiceContext) *RetryKbDocLogic {
	return &RetryKbDocLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *RetryKbDocLogic) RetryKbDoc(in *aipb.RetryKbDocRequest) (*aipb.RetryKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	r := model.KbEnqueueRequest{KbId: in.KbId, DocId: in.DocId, OwnerId: in.Actor.UserId, Operation: "retry", RequestId: in.RequestId, BatchId: in.BatchId, EmbeddingModel: l.svcCtx.Config.AI.EmbeddingModel}
	out, err := findKbReplay(l.ctx, l.svcCtx, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		doc, err := l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
		if err != nil {
			return nil, aiModelError(err)
		}
		if err := requireKbReady(l.svcCtx); err != nil {
			return nil, err
		}
		r.ExpectedVersion = doc.Version
		out, err = enqueueKbDoc(l.ctx, l.svcCtx, r, chunkText(doc.Content))
		if err != nil {
			return nil, err
		}
	}
	return &aipb.RetryKbDocResponse{Doc: kbDocRowToPb(out.Doc), JobId: out.Job.Id, BatchId: out.BatchId}, nil
}
