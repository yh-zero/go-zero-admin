package aiservicelogic

import (
	"context"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateKbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateKbLogic {
	return &CreateKbLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateKbLogic) CreateKb(in *aipb.CreateKbRequest) (*aipb.CreateKbResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" || utf8.RuneCountInString(name) > 128 {
		return nil, xerr.NewErrCodeMsg(400, "知识库名称须为 1..128 字符")
	}
	desc := strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(desc) > 512 {
		return nil, xerr.NewErrCodeMsg(400, "知识库描述最多 512 字符")
	}
	result, err := l.svcCtx.AiKbModel.Insert(l.ctx, &model.AiKb{OwnerId: uint64(in.Actor.UserId), Name: name, Description: desc})
	if err != nil {
		return nil, aiModelError(err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, aiModelError(err)
	}
	// 回读权威行，保证响应时间戳等字段与库一致
	row, err := l.svcCtx.AiKbModel.FindOwned(l.ctx, id, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.CreateKbResponse{Kb: kbRowToPb(row), Capabilities: capabilitiesProto(l.svcCtx)}, nil
}
