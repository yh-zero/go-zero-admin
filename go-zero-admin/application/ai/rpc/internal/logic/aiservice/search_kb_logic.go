package aiservicelogic

import (
	"context"
	"github.com/google/uuid"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchKbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSearchKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchKbLogic {
	return &SearchKbLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SearchKbLogic) SearchKb(in *aipb.SearchKbRequest) (*aipb.SearchKbResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.KbId <= 0 || strings.TrimSpace(in.Query) == "" || !utf8.ValidString(in.Query) || len(in.Query) > 2000 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()
	// 先校验归属，避免越权请求触发向量化，也不向其透露检索配置。
	if _, err := l.svcCtx.AiKbModel.FindOwned(ctx, in.KbId, in.Actor.UserId); err != nil {
		return nil, aiModelError(err)
	}
	if l.svcCtx.KbSearcher == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "知识库检索未配置，请联系管理员")
	}
	started := time.Now()
	ctx = svc.WithUsageSource(ctx, in.Actor.UserId, "kb_search", uuid.NewString(), in.KbId, -1)
	hits, err := l.svcCtx.KbSearcher.Search(ctx, in.Actor.UserId, in.KbId, in.Query)
	if err != nil || ctx.Err() != nil {
		if code, ok := xerr.AsCodeError(err); ok && (code.GetErrCode() == 429 || code.GetErrCode() == 503) {
			return nil, code
		}
		l.Error("知识库召回失败或超时")
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "知识库检索暂不可用，请稍后重试")
	}
	list := make([]*aipb.KbSearchHit, 0, len(hits))
	for _, hit := range hits {
		list = append(list, &aipb.KbSearchHit{ChunkId: hit.ChunkID, KbId: hit.KbID, DocId: hit.DocID, DocTitle: hit.DocTitle, Snippet: hit.Snippet, Score: hit.Score})
	}
	return &aipb.SearchKbResponse{List: list, ElapsedMs: time.Since(started).Milliseconds()}, nil
}
