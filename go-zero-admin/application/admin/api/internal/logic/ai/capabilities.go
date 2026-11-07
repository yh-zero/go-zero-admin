package ai

import (
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

func capabilitiesInfo(value *aipb.Capabilities) *types.AiCapabilities {
	if value == nil {
		return nil
	}
	return &types.AiCapabilities{RetrievalConfigured: value.RetrievalConfigured, AnalysisConfigured: value.AnalysisConfigured, IngestionMode: value.IngestionMode}
}
