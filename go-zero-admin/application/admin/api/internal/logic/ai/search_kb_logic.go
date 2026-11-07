// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ai

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SearchKbLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSearchKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SearchKbLogic {
	return &SearchKbLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SearchKbLogic) SearchKb(req *types.AiSearchKbReq) (*types.AiSearchKbResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if strings.TrimSpace(req.Query) == "" || !utf8.ValidString(req.Query) {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(l.ctx, 10*time.Second)
	defer cancel()
	out, err := l.svcCtx.AiRpc.SearchKb(ctx, &aipb.SearchKbRequest{Actor: actor, KbId: req.Id, Query: req.Query})
	if err != nil {
		if _, ok := xerr.AsCodeError(err); ok {
			return nil, err
		}
		if _, _, ok := xerr.FromGRPCError(err); ok {
			return nil, err
		}
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "知识库检索暂不可用，请稍后重试")
	}
	list := make([]*types.AiKbSearchHit, 0, len(out.List))
	for _, hit := range out.List {
		list = append(list, &types.AiKbSearchHit{ChunkId: hit.ChunkId, KbId: hit.KbId, DocId: hit.DocId, DocTitle: hit.DocTitle, Snippet: hit.Snippet, Score: hit.Score})
	}
	return &types.AiSearchKbResp{List: list, ElapsedMs: out.ElapsedMs}, nil
}
