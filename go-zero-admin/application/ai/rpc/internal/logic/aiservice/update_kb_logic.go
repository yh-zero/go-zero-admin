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

type UpdateKbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateKbLogic {
	return &UpdateKbLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateKbLogic) UpdateKb(in *aipb.UpdateKbRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	patch := model.KbPatch{}
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" || utf8.RuneCountInString(name) > 128 {
			return nil, xerr.NewErrCodeMsg(400, "知识库名称须为 1..128 字符")
		}
		patch.Name = &name
	}
	if in.Description != nil {
		desc := strings.TrimSpace(*in.Description)
		if utf8.RuneCountInString(desc) > 512 {
			return nil, xerr.NewErrCodeMsg(400, "知识库描述最多 512 字符")
		}
		patch.Description = &desc
	}
	if err := l.svcCtx.AiKbModel.UpdateOwned(l.ctx, in.KbId, in.Actor.UserId, patch); err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}
