// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package ai

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetMessageFeedbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSetMessageFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMessageFeedbackLogic {
	return &SetMessageFeedbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetMessageFeedbackLogic) SetMessageFeedback(req *types.AiMessageFeedbackReq) (resp *types.EmptyResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	if _, err := l.svcCtx.AiRpc.SetMessageFeedback(l.ctx, &aipb.SetMessageFeedbackRequest{Actor: actor, ConversationId: req.Id, MessageId: req.MessageId, Feedback: req.Feedback, Note: req.Note}); err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
