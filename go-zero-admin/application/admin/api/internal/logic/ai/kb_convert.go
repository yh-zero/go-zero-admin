package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func mapAiKb(row *aipb.KbInfo) *types.AiKbInfo {
	return &types.AiKbInfo{
		Id: row.Id, Name: row.Name, Description: row.Description, OwnerId: row.OwnerId,
		DocCount: row.DocCount, ChunkCount: row.ChunkCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		ReadyCount: row.ReadyCount, PendingCount: row.PendingCount, FailedCount: row.FailedCount,
	}
}

func mapAiKbDoc(row *aipb.KbDocInfo) *types.AiKbDocInfo {
	if row == nil {
		return nil
	}
	return &types.AiKbDocInfo{
		Id: row.Id, KbId: row.KbId, Title: row.Title, Status: row.Status, Error: row.Error,
		ChunkCount: row.ChunkCount, EmbeddingModel: row.EmbeddingModel, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
		Version: row.Version, Stage: row.Stage,
	}
}

func mapAiKbBatch(r *aipb.KbIngestBatchInfo) *types.AiKbIngestBatchInfo {
	return &types.AiKbIngestBatchInfo{Id: r.Id, KbId: r.KbId, Stopped: r.Stopped, Total: r.Total, Queued: r.Queued, Running: r.Running, Succeeded: r.Succeeded, Failed: r.Failed, Cancelled: r.Cancelled, Reused: r.Reused, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func mapAiKbImport(out *aipb.GetKbImportResponse) *types.AiKbImportResp {
	list := make([]*types.AiKbIngestJobInfo, 0, len(out.List))
	for _, r := range out.List {
		list = append(list, &types.AiKbIngestJobInfo{Id: r.Id, RequestId: r.RequestId, DocId: r.DocId, Operation: r.Operation, Status: r.Status, Error: r.Error, Doc: mapAiKbDoc(r.Doc), CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return &types.AiKbImportResp{Batch: mapAiKbBatch(out.Batch), List: list}
}
