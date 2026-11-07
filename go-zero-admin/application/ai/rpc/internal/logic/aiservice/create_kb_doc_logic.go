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

type CreateKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateKbDocLogic(ctx context.Context, s *svc.ServiceContext) *CreateKbDocLogic {
	return &CreateKbDocLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *CreateKbDocLogic) CreateKbDoc(in *aipb.CreateKbDocRequest) (*aipb.CreateKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || !utf8.ValidString(title) || utf8.RuneCountInString(title) > 128 || !utf8.ValidString(in.Content) || len(in.Content) == 0 || len(in.Content) > kbContentMaxBytes {
		return nil, xerr.NewErrCodeMsg(400, "文档标题须为1..128字符，正文须为1..65536 UTF-8字节")
	}
	r := model.KbEnqueueRequest{KbId: in.KbId, OwnerId: in.Actor.UserId, Operation: "create", RequestId: in.RequestId, BatchId: in.BatchId, Title: title, Content: in.Content, EmbeddingModel: l.svcCtx.Config.AI.EmbeddingModel}
	out, err := findKbReplay(l.ctx, l.svcCtx, r)
	if err != nil {
		return nil, err
	}
	if out == nil {
		if _, err := l.svcCtx.AiKbModel.FindOwned(l.ctx, in.KbId, in.Actor.UserId); err != nil {
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
	return &aipb.CreateKbDocResponse{Doc: kbDocRowToPb(out.Doc), JobId: out.Job.Id, BatchId: out.BatchId}, nil
}
