package aiservicelogic

import (
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/model"
)

func kbRowToPb(r *model.KbRow) *aipb.KbInfo {
	return &aipb.KbInfo{
		Id:          r.Id,
		Name:        r.Name,
		Description: r.Description,
		OwnerId:     r.OwnerId,
		DocCount:    r.DocCount,
		ChunkCount:  r.ChunkCount,
		ReadyCount:  r.ReadyCount, PendingCount: r.PendingCount, FailedCount: r.FailedCount,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

func kbDocRowToPb(r *model.KbDocRow) *aipb.KbDocInfo {
	return &aipb.KbDocInfo{
		Id:             r.Id,
		KbId:           r.KbId,
		Title:          r.Title,
		Status:         r.Status,
		Error:          r.Error,
		ChunkCount:     r.ChunkCount,
		EmbeddingModel: r.EmbeddingModel,
		CreatedAt:      r.CreatedAt,
		UpdatedAt:      r.UpdatedAt,
		Version:        r.Version,
		Stage:          kbDocumentStage(r),
	}
}

func kbDocumentStage(r *model.KbDocRow) string {
	if r.Status == "ready" || r.Status == "failed" {
		return r.Status
	}
	if r.Stage == "indexing" || r.Stage == "queued" {
		return r.Stage
	}
	return "embedding"
}

func kbBatchToPb(r *model.KbIngestBatchRow) *aipb.KbIngestBatchInfo {
	return &aipb.KbIngestBatchInfo{Id: r.Id, KbId: r.KbId, Stopped: r.Stopped, Total: r.Total, Queued: r.Queued, Running: r.Running, Succeeded: r.Succeeded, Failed: r.Failed, Cancelled: r.Cancelled, Reused: r.Reused, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

func kbImportToPb(batch *model.KbIngestBatchRow, jobs []*model.KbIngestJobView) *aipb.GetKbImportResponse {
	list := make([]*aipb.KbIngestJobInfo, 0, len(jobs))
	for _, r := range jobs {
		row := &aipb.KbIngestJobInfo{Id: r.Id, RequestId: r.RequestId, DocId: r.DocId, Operation: r.Operation, Status: r.Status, Error: r.Error, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
		if r.Doc != nil {
			row.Doc = kbDocRowToPb(r.Doc)
		}
		list = append(list, row)
	}
	return &aipb.GetKbImportResponse{Batch: kbBatchToPb(batch), List: list}
}
