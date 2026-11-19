package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/xerr"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListKbDocsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbDocsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbDocsLogic {
	return &ListKbDocsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListKbDocsLogic) ListKbDocs(in *aipb.ListKbDocsRequest) (*aipb.ListKbDocsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(in.Query) || utf8.RuneCountInString(in.Query) > 128 || (in.Status != "" && in.Status != "ready" && in.Status != "pending" && in.Status != "failed") {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	result, err := l.svcCtx.AiKbDocModel.ReadPage(l.ctx, in.KbId, in.Actor.UserId, page, size, strings.TrimSpace(in.Query), in.Status, in.MetaOnly)
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.KbDocInfo, 0, len(result.Rows))
	for _, r := range result.Rows {
		list = append(list, kbDocRowToPb(r))
	}
	return &aipb.ListKbDocsResponse{List: list, Page: &aipb.PageResponse{Total: result.Total}, ReadyCount: result.Ready, PendingCount: result.Pending, FailedCount: result.Failed, PageSignature: result.Signature}, nil
}
