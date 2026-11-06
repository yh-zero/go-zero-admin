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

type CreateKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateKbDocLogic {
	return &CreateKbDocLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// CreateKbDoc 创建文档并同步摄取（契约：摄取在请求内完成并返回最终 status）
func (l *CreateKbDocLogic) CreateKbDoc(in *aipb.CreateKbDocRequest) (*aipb.CreateKbDocResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" || utf8.RuneCountInString(title) > 128 {
		return nil, xerr.NewErrCodeMsg(400, "文档标题须为 1..128 字符")
	}
	if len(in.Content) == 0 || len(in.Content) > kbContentMaxBytes {
		return nil, xerr.NewErrCodeMsg(400, "文档内容须为 1..65536 字节")
	}
	// 归属/存在性校验先于就绪校验：404 语义不受降级影响
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
	docId, chunkIds, err := l.svcCtx.AiKbDocModel.InsertWithChunks(l.ctx, &model.KbDocRow{
		KbId:           in.KbId,
		Title:          title,
		Content:        in.Content,
		EmbeddingModel: l.svcCtx.Config.AI.EmbeddingModel,
	}, chunks)
	if err != nil {
		return nil, aiModelError(err)
	}
	if err := ingestChunks(l.ctx, l.svcCtx, docId, in.KbId, in.Actor.UserId, chunks, chunkIds); err != nil {
		l.Errorf("文档 %d 摄取失败: %v", docId, err)
	}
	row, err := l.svcCtx.AiKbDocModel.FindOwned(l.ctx, in.KbId, docId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.CreateKbDocResponse{Doc: kbDocRowToPb(row)}, nil
}
