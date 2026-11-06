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
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
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
	}
}
