package aiservicelogic

import (
	"context"
	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

const kbContentMaxBytes = 64 * 1024
const kbIngestTimeout = svc.KbIngestTimeout

func ingestChunks(ctx context.Context, s *svc.ServiceContext, docId, kbId, ownerId int64, chunks []model.KbChunkRow, ids []uint64) error {
	return s.IngestKbChunks(ctx, docId, kbId, ownerId, chunks, ids)
}

func requireKbReady(s *svc.ServiceContext) error {
	if s.Embedder == nil || s.VectorStore == nil {
		return kbQueueError(model.ErrKbIngestUnavailable)
	}
	return nil
}
