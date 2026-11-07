package aiservicelogic

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListConversationsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListConversationsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListConversationsLogic {
	return &ListConversationsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListConversationsLogic) ListConversations(in *aipb.ListConversationsRequest) (*aipb.ListConversationsResponse, error) {
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
	rows, total, err := l.svcCtx.AiConversationModel.SearchOwned(l.ctx, in.Actor.UserId, page, size, strings.TrimSpace(in.Query))
	if err != nil {
		return nil, aiModelError(err)
	}
	list := make([]*aipb.ConversationInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, conversationProto(row))
	}
	return &aipb.ListConversationsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
