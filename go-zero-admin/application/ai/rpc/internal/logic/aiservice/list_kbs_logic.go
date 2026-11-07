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

type ListKbsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListKbsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListKbsLogic {
	return &ListKbsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListKbsLogic) ListKbs(in *aipb.ListKbsRequest) (*aipb.ListKbsResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	page, size, err := aiPagination(in.Page)
	if err != nil {
		return nil, err
	}
	if !utf8.ValidString(in.Query) || utf8.RuneCountInString(in.Query) > 128 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	rows, total, err := l.svcCtx.AiKbModel.SearchOwned(l.ctx, in.Actor.UserId, page, size, strings.TrimSpace(in.Query))
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.KbInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, kbRowToPb(r))
	}
	return &aipb.ListKbsResponse{List: list, Page: &aipb.PageResponse{Total: total}, Capabilities: capabilitiesProto(l.svcCtx)}, nil
}
