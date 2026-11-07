package aiservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"strings"
	"unicode/utf8"
)

type UpdateKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateKbDocLogic(ctx context.Context, s *svc.ServiceContext) *UpdateKbDocLogic {
	return &UpdateKbDocLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *UpdateKbDocLogic) UpdateKbDoc(in *aipb.UpdateKbDocRequest) (*aipb.UpdateKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || !utf8.ValidString(title) || utf8.RuneCountInString(title) > 128 || !utf8.ValidString(in.Content) || len(in.Content) == 0 || len(in.Content) > kbContentMaxBytes || in.ExpectedVersion < 1 {
		return nil, xerr.NewErrCodeMsg(400, "文档标题、正文或当前版本无效")
	}
	r := model.KbEnqueueRequest{KbId: in.KbId, DocId: in.DocId, OwnerId: in.Actor.UserId, ExpectedVersion: in.ExpectedVersion, Operation: "update", RequestId: in.RequestId, BatchId: in.BatchId, Title: title, Content: in.Content, EmbeddingModel: l.svcCtx.Config.AI.EmbeddingModel}
	out, err := findKbReplay(l.ctx, l.svcCtx, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		if _, err := l.svcCtx.AiKbDocModel.FindMetadataOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId); err != nil {
			return nil, aiModelError(err)
		}
		if err := requireKbReady(l.svcCtx); err != nil {
			return nil, err
		}
		chunks := chunkText(in.Content)
		if len(chunks) == 0 {
			return nil, xerr.NewErrCodeMsg(400, "文档内容无法分块（全为空白）")
		}
		out, err = enqueueKbDoc(l.ctx, l.svcCtx, r, chunks)
		if err != nil {
			return nil, err
		}
	}
	return &aipb.UpdateKbDocResponse{Doc: kbDocRowToPb(out.Doc), JobId: out.Job.Id, BatchId: out.BatchId}, nil
}
