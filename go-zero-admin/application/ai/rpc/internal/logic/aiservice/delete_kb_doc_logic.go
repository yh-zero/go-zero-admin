package aiservicelogic

import (
	"context"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteKbDocLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteKbDocLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteKbDocLogic {
	return &DeleteKbDocLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteKbDocLogic) DeleteKbDoc(in *aipb.DeleteKbDocRequest) (*aipb.NoDataResponse, error) {
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	chunkIds, err := l.svcCtx.AiKbDocModel.DeleteOwned(l.ctx, in.KbId, in.DocId, in.Actor.UserId)
	if err != nil && !model.KbDocCommitted(err) {
		return nil, aiModelError(err)
	}
	if err != nil {
		l.Errorf("文档 %d 删除已提交但缓存失效失败: %v", in.DocId, err)
	}
	if l.svcCtx.VectorStore != nil {
		if err := l.svcCtx.VectorStore.Delete(l.ctx, chunkIds); err != nil {
			l.Errorf("文档 %d 向量索引清除失败（等待对账收敛）: %v", in.DocId, err)
		}
	}
	if err != nil {
		return nil, aiModelError(err)
	}
	return &aipb.NoDataResponse{}, nil
}
