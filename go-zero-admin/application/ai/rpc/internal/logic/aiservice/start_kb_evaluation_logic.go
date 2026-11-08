package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type StartKbEvaluationLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewStartKbEvaluationLogic(ctx context.Context, svcCtx *svc.ServiceContext) *StartKbEvaluationLogic {
	return &StartKbEvaluationLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *StartKbEvaluationLogic) StartKbEvaluation(in *aipb.StartKbEvaluationRequest) (*aipb.KbEvaluationResponse, error) {
	if err := requireAiActor(in.GetActor()); err != nil {
		return nil, err
	}
	signature, err := evaluationRequestSignature(in)
	if err != nil {
		return nil, err
	}
	if err = checkEvaluationKb(l.ctx, l.svcCtx, in.Actor, in.KbId); err != nil {
		return nil, err
	}
	if existing, readErr := l.svcCtx.AiKbEvaluationModel.FindRequestOwned(l.ctx, in.Actor.UserId, in.RequestId); readErr == nil {
		if existing.KbId != in.KbId || existing.RequestSignature != signature {
			return nil, evaluationError(model.ErrEvaluationConflict)
		}
		existing, err = refreshEvaluationExpiry(l.ctx, l.svcCtx, existing)
		if err != nil {
			return nil, evaluationError(err)
		}
		return &aipb.KbEvaluationResponse{Evaluation: evaluationProto(existing, false)}, nil
	} else if !errors.Is(readErr, model.ErrNotFound) {
		return nil, evaluationError(readErr)
	}
	var dataset *evaluationDataset
	if in.DatasetSnapshot != "" {
		if !strings.HasPrefix(in.DatasetKey, "custom-") {
			return nil, xerr.NewErrCodeMsg(400, "自定义快照须使用对应的custom题集标识")
		}
		sum := sha256.Sum256([]byte(in.DatasetSnapshot))
		if hex.EncodeToString(sum[:]) != in.DatasetSha256 {
			return nil, xerr.NewErrCodeMsg(409, "题集原字节指纹不一致，请重新核对")
		}
		dataset, err = parseCustomEvaluationDataset(in.DatasetKey, in.DatasetSha256, in.DatasetSnapshot)
	} else {
		dataset, err = loadEvaluationDataset(in.DatasetKey)
	}
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "请选择有效题集或提供符合规范的自定义快照")
	}
	if dataset.SHA != in.DatasetSha256 {
		return nil, xerr.NewErrCodeMsg(409, "题集版本已变化，请刷新后核对")
	}
	cases, err := selectEvaluationCases(dataset, in.GroupKey, in.CaseIds)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "所选题目或知识库分组无效")
	}
	if l.svcCtx.KbSearcher == nil || l.svcCtx.LLM == nil || !json.Valid([]byte(l.svcCtx.EvaluationContext)) {
		return nil, xerr.NewErrCodeMsg(503, "知识库检索、分析模型或评测元数据未配置")
	}
	if _, ok := agenttools.Get("kb_analyze"); !ok {
		return nil, xerr.NewErrCodeMsg(503, "知识库分析能力暂不可用")
	}
	corpus, err := l.svcCtx.AiKbEvaluationModel.ReadCorpusOwned(l.ctx, in.Actor.UserId, in.KbId)
	if err != nil {
		return nil, evaluationError(err)
	}
	bindings, err := buildEvaluationBindings(dataset, in.GroupKey, in.KbId, corpus)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(409, "当前知识库的文档标题、正文版本或就绪状态与题集资料不匹配")
	}
	observations := make([]model.EvaluationObservation, 0, len(cases))
	for _, item := range cases {
		observations = append(observations, model.EvaluationObservation{CaseId: item.Id, RequestId: uuid.NewString(), Query: item.Query, Status: "waiting", Stage: "waiting", List: json.RawMessage(`[]`)})
	}
	raw, _ := json.Marshal(observations)
	if !reserveEvaluationSlot() {
		return nil, xerr.NewErrCodeMsg(503, "评测执行槽已满，请稍后使用原请求标识核对")
	}
	row, created, err := l.svcCtx.AiKbEvaluationModel.CreateOwned(l.ctx, &model.KbEvaluationRow{OwnerId: in.Actor.UserId, KbId: in.KbId, RequestId: in.RequestId, RequestSignature: signature, DatasetKey: in.DatasetKey, DatasetSHA: dataset.SHA, GroupKey: in.GroupKey, CaseCount: int64(len(cases)), Snapshot: dataset.Raw, Bindings: bindings, Context: l.svcCtx.EvaluationContext, Observations: string(raw)})
	if created && row != nil {
		launchEvaluation(l.ctx, l.svcCtx, in.Actor, row)
	} else {
		releaseEvaluationSlot()
	}
	if err != nil {
		return nil, evaluationError(err)
	}
	return &aipb.KbEvaluationResponse{Evaluation: evaluationProto(row, false)}, nil
}
