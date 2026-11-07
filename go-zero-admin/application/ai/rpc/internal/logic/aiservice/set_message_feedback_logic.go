package aiservicelogic

import (
	"context"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetMessageFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetMessageFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetMessageFeedbackLogic {
	return &SetMessageFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetMessageFeedbackLogic) SetMessageFeedback(in *aipb.SetMessageFeedbackRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	if in.ConversationId <= 0 || in.MessageId <= 0 || !validMessageFeedback(in.Feedback) || !utf8.ValidString(in.Note) || len(in.Note) > 1000 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if in.Feedback == "" {
		in.Note = ""
	}
	if err := l.svcCtx.AiMessageModel.SetFeedbackOwned(l.ctx, in.Actor.UserId, in.ConversationId, in.MessageId, in.Feedback, in.Note); err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}

func validMessageFeedback(value string) bool {
	return value == "" || value == "helpful" || value == "irrelevant_source" || value == "missing_knowledge"
}
