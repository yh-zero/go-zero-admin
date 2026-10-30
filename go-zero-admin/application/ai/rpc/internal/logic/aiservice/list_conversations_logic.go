package aiservicelogic

import (
	"context"

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
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total,
		"SELECT COUNT(*) FROM ai_conversation WHERE user_id=? AND deleted_at IS NULL", in.Actor.UserId)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	type convRow struct {
		Id           int64  `db:"id"`
		Title        string `db:"title"`
		MessageCount int64  `db:"message_count"`
		CreatedAt    int64  `db:"created_at"`
		UpdatedAt    int64  `db:"updated_at"`
	}
	var rows []convRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT c.id, c.title, (SELECT COUNT(*) FROM ai_message m WHERE m.conversation_id=c.id), UNIX_TIMESTAMP(c.created_at), UNIX_TIMESTAMP(c.updated_at) FROM ai_conversation c WHERE c.user_id=? AND c.deleted_at IS NULL ORDER BY c.updated_at DESC LIMIT ? OFFSET ?",
		in.Actor.UserId, pageSize, offset)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*aipb.ConversationInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &aipb.ConversationInfo{Id: r.Id, Title: r.Title, UserId: in.Actor.UserId, MessageCount: r.MessageCount, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return &aipb.ListConversationsResponse{List: list, Page: &aipb.PageResponse{Total: total}}, nil
}
