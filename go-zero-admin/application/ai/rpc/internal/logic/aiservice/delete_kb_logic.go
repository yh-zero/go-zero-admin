package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteKbLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteKbLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKbLogic {
	return &DeleteKbLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteKbLogic) DeleteKb(in *aipb.DeleteKbRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	// 事实源先删（事务内软删 kb/文档、删 chunk 行）；向量索引后删，失败时检索侧有 MySQL 复核兜底
	chunkIds, err := l.svcCtx.AiKbModel.DeleteOwned(l.ctx, in.KbId, in.Actor.UserId)
	if err != nil {
		return nil, aiModelError(err)
	}
	if l.svcCtx.VectorStore != nil {
		if err := l.svcCtx.VectorStore.Delete(l.ctx, chunkIds); err != nil {
			l.Errorf("知识库 %d 向量索引清除失败（等待对账收敛）: %v", in.KbId, err)
		}
	}
	return &aipb.NoDataResponse{}, nil
}
