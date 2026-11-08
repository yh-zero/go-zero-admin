package aiservicelogic

import (
	"context"
	"encoding/json"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetTaskFeedbackLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetTaskFeedbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetTaskFeedbackLogic {
	return &SetTaskFeedbackLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetTaskFeedbackLogic) SetTaskFeedback(in *aipb.SetTaskFeedbackRequest) (*aipb.KbFeedbackResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	if in.TaskId <= 0 || in.StepIndex < 0 || in.StepIndex > 31 || in.ExpectedRevision < 0 || !validFeedbackText(in.Note, 1000, false) {
		return nil, xerr.NewErrCode(400)
	}
	switch in.Feedback {
	case "helpful", "incorrect_answer", "missing_knowledge", "irrelevant_source":
	default:
		return nil, xerr.NewErrCode(400)
	}
	task, err := l.svcCtx.AiTaskModel.FindOwned(l.ctx, in.TaskId, in.Actor.UserId)
	if err != nil {
		return nil, feedbackError(err)
	}
	snapshot, step, answer, err := analysisFeedbackSnapshot(task, in.StepIndex)
	if err != nil {
		return nil, err
	}
	if _, err = l.svcCtx.AiKbModel.FindOwned(l.ctx, snapshot.KbId, in.Actor.UserId); err != nil {
		return nil, feedbackError(err)
	}
	if err = captureFeedbackDocuments(l.ctx, l.svcCtx, in.Actor.UserId, snapshot); err != nil {
		return nil, err
	}
	raw, err := json.Marshal(snapshot)
	if err != nil || len(raw) > 64000 {
		return nil, xerr.NewErrCodeMsg(400, "反馈快照超出可保存范围")
	}
	row, err := l.svcCtx.AiKbFeedbackModel.SetOwned(l.ctx, in.Actor.UserId, in.ExpectedRevision, &model.KbFeedbackRow{KbId: snapshot.KbId, TaskId: in.TaskId, StepIndex: in.StepIndex, Feedback: in.Feedback, Note: in.Note, Question: snapshot.Query, Answer: answer, Snapshot: string(raw)}, step)
	if err != nil {
		return nil, feedbackError(err)
	}
	item, err := kbFeedbackProto(row)
	return &aipb.KbFeedbackResponse{Feedback: item}, err
}
