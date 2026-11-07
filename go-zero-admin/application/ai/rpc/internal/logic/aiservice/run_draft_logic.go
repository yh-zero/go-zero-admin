package aiservicelogic

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type RunDraftLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRunDraftLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RunDraftLogic {
	return &RunDraftLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RunDraftLogic) RunDraft(in *aipb.RunDraftRequest) (*aipb.RunWorkflowResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return nil, xerr.NewErrCodeMsg(400, "流程用途须为1..128字符")
	}
	input, err := normalizeWorkflowInput(in.Input)
	if err != nil {
		return nil, err
	}
	parsed, err := uuid.Parse(in.RequestId)
	if err != nil || parsed.String() != in.RequestId {
		return nil, xerr.NewErrCodeMsg(400, "request_id 必须为规范 UUID")
	}
	var header struct {
		Steps []json.RawMessage `json:"steps"`
	}
	if len(in.Definition) == 0 || len(in.Definition) > 65536 || json.Unmarshal([]byte(in.Definition), &header) != nil || len(header.Steps) == 0 || len(header.Steps) > 32 {
		return nil, xerr.NewErrCodeMsg(400, "工作流须包含1..32个有效步骤且最多65536字节")
	}
	definitionValue, err := readWorkflowJSON(in.Definition)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "工作流定义无效")
	}
	inputValue, _ := readWorkflowJSON(input)
	identity, _ := json.Marshal(map[string]any{"name": name, "definition": definitionValue, "input": inputValue})
	digest := sha256.Sum256(identity)
	signature := hex.EncodeToString(digest[:])
	// 已接收任务的重查不依赖工具仍被注册，也不占新执行槽。
	if existing, readErr := l.svcCtx.AiWorkflowModel.FindTaskRequestOwned(l.ctx, in.Actor.UserId, in.RequestId); readErr == nil {
		if existing.WorkflowId != 0 || existing.RequestSignature != signature {
			return nil, aiModelError(model.ErrTaskRequestConflict)
		}
		return &aipb.RunWorkflowResponse{TaskId: existing.Id, Status: existing.Status}, nil
	} else if !errors.Is(readErr, model.ErrNotFound) {
		return nil, aiModelError(readErr)
	}
	if _, err := parseWorkflow(in.Definition); err != nil {
		return nil, err
	}
	if !reserveWorkflowSlot() {
		return nil, xerr.NewErrCodeMsg(503, "工作流繁忙，请稍后使用同一请求标识核对")
	}
	task, created, err := l.svcCtx.AiWorkflowModel.CreateDraftTaskIdempotent(l.ctx, in.Actor.UserId, name, in.Definition, input, in.RequestId, signature)
	if err != nil {
		if created && task != nil && errors.Is(err, model.ErrTaskCommitted) {
			launchWorkflow(l.ctx, l.svcCtx, task, false)
			return nil, xerr.NewErrCodeMsg(503, "任务已创建，缓存刷新失败，请使用同一请求标识核对")
		}
		releaseWorkflowSlot()
		return nil, aiModelError(err)
	}
	if created {
		launchWorkflow(l.ctx, l.svcCtx, task, false)
	} else {
		releaseWorkflowSlot()
	}
	return &aipb.RunWorkflowResponse{TaskId: task.Id, Status: task.Status}, nil
}

func normalizeWorkflowInput(input string) (string, error) {
	if input == "" {
		input = "{}"
	}
	if len(input) > 16000 || !json.Valid([]byte(input)) {
		return "", xerr.NewErrCodeMsg(400, "input 必须为最多16000字节的JSON对象")
	}
	var object map[string]json.RawMessage
	if json.Unmarshal([]byte(input), &object) != nil || object == nil {
		return "", xerr.NewErrCodeMsg(400, "运行输入必须为JSON对象")
	}
	canonical, _ := json.Marshal(object)
	return string(canonical), nil
}

func normalizeWorkspaceQuery(query string) (string, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > 128 {
		return "", xerr.NewErrCodeMsg(400, "查找条件最多128字符")
	}
	return query, nil
}
