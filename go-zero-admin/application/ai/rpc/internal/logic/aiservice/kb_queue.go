package aiservicelogic

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

func validateKbRequestIds(requestId, batchId string) error {
	ids := []string{requestId}
	if batchId != "" {
		ids = append(ids, batchId)
	}
	for _, id := range ids {
		parsed, err := uuid.Parse(id)
		if err != nil || parsed.String() != id {
			return xerr.NewErrCodeMsg(400, "request_id 和非空 batch_id 必须为规范 UUID")
		}
	}
	return nil
}

func kbQueueError(err error) error {
	switch {
	case errors.Is(err, model.ErrKbRequestConflict):
		return xerr.NewErrCodeMsg(400, "请求标识已用于其他内容，请核对原请求")
	case errors.Is(err, model.ErrKbBatchStopped):
		return xerr.NewErrCodeMsg(400, "该批次已停止，不能追加文档")
	case errors.Is(err, model.ErrKbBatchFull):
		return xerr.NewErrCodeMsg(400, "每个批次最多接收20项")
	default:
		return aiModelError(err)
	}
}

func findKbReplay(ctx context.Context, s *svc.ServiceContext, r model.KbEnqueueRequest) (*model.KbEnqueueResult, error) {
	if err := validateKbRequestIds(r.RequestId, r.BatchId); err != nil {
		return nil, err
	}
	out, err := s.KbIngestQueue.FindRequest(ctx, r)
	if errors.Is(err, model.ErrNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, kbQueueError(err)
	}
	return out, nil
}

func enqueueKbDoc(ctx context.Context, s *svc.ServiceContext, r model.KbEnqueueRequest, chunks []model.KbChunkRow) (*model.KbEnqueueResult, error) {
	out, err := s.KbIngestQueue.Enqueue(ctx, r, chunks)
	if err != nil && !model.KbDocCommitted(err) {
		return nil, kbQueueError(err)
	}
	return out, nil
}
