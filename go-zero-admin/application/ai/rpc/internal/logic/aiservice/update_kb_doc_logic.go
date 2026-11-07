package aiservicelogic

import (
	"context"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateKbDocLogic {
	return &UpdateKbDocLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateKbDocLogic) UpdateKbDoc(in *aipb.UpdateKbDocRequest) (*aipb.UpdateKbDocResponse, error) {
	ctx, cancel := context.WithTimeout(l.ctx, kbIngestTimeout)
	defer cancel()
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > 128 || !utf8.ValidString(title) {
		return nil, xerr.NewErrCodeMsg(400, "文档标题须为1..128字符")
	}
	if len(in.Content) == 0 || len(in.Content) > kbContentMaxBytes || !utf8.ValidString(in.Content) || in.ExpectedVersion < 1 {
		return nil, xerr.NewErrCodeMsg(400, "文档内容须为1..65536 UTF-8字节，且须提供当前版本")
	}
	if _, err := l.svcCtx.AiKbDocModel.FindMetadataOwned(ctx, in.KbId, in.DocId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	if err := requireKbReady(l.svcCtx); err != nil {
		return nil, err
	}
	chunks := chunkText(in.Content)
	if len(chunks) == 0 {
		return nil, xerr.NewErrCodeMsg(400, "文档内容无法分块（全为空白）")
	}
	oldIds, nextIds, err := l.svcCtx.AiKbDocModel.ReplaceDocument(ctx, in.Actor.UserId, in.ExpectedVersion, &model.KbDocRow{
		Id: in.DocId, KbId: in.KbId, Title: title, Content: in.Content, EmbeddingModel: l.svcCtx.Config.AI.EmbeddingModel,
	}, chunks)
	if err != nil && !model.KbDocCommitted(err) {
		if err == model.ErrKbDocState {
			return nil, xerr.NewErrCodeMsg(400, "文档已被修改，请刷新原文和版本后重新编辑")
		}
		return nil, aiModelError(err)
	}
	if err != nil {
		l.Errorf("文档 %d 更新已提交但缓存失效失败: %v", in.DocId, err)
	}
	if len(oldIds) > 0 {
		if err := l.svcCtx.VectorStore.Delete(ctx, oldIds); err != nil {
			l.Errorf("文档 %d 更新后旧向量清除失败，需对账: %v", in.DocId, err)
		}
	}
	if err := ingestChunks(ctx, l.svcCtx, in.DocId, in.KbId, in.Actor.UserId, chunks, nextIds); err != nil {
		l.Errorf("文档 %d 更新后摄取失败: %v", in.DocId, err)
	}
	row, err := l.svcCtx.AiKbDocModel.FindMetadataOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.UpdateKbDocResponse{Doc: kbDocRowToPb(row)}, nil
}
