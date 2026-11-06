package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func mapAiKb(row *aipb.KbInfo) *types.AiKbInfo {
	return &types.AiKbInfo{
		Id: row.Id, Name: row.Name, Description: row.Description, OwnerId: row.OwnerId,
		DocCount: row.DocCount, ChunkCount: row.ChunkCount, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapAiKbDoc(row *aipb.KbDocInfo) *types.AiKbDocInfo {
	return &types.AiKbDocInfo{
		Id: row.Id, KbId: row.KbId, Title: row.Title, Status: row.Status, Error: row.Error,
		ChunkCount: row.ChunkCount, EmbeddingModel: row.EmbeddingModel, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}
