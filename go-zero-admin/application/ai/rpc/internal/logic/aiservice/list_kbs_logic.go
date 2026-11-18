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
	if !utf8.ValidString(in.Query) || utf8.RuneCountInString(in.Query) > 128 || in.Id < 0 || in.Id != 0 && !in.ChoicesOnly {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if in.ChoicesOnly {
		rows, total, err := l.svcCtx.AiKbModel.ChoicesOwned(l.ctx, in.Actor.UserId, in.Id, page, size, strings.TrimSpace(in.Query))
		if err != nil {
			return nil, aiModelError(err)
		}
		choices := make([]*aipb.KbChoice, 0, len(rows))
		for _, row := range rows {
			choices = append(choices, &aipb.KbChoice{Id: row.Id, Name: row.Name})
		}
		return &aipb.ListKbsResponse{Choices: choices, Page: &aipb.PageResponse{Total: total}}, nil
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
