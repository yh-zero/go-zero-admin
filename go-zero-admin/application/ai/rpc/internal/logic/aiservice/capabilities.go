package aiservicelogic

import (
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

func capabilitiesProto(service *svc.ServiceContext) *aipb.Capabilities {
	retrieval := service != nil && service.KbSearcher != nil
	return &aipb.Capabilities{RetrievalConfigured: retrieval, AnalysisConfigured: retrieval && service.LLM != nil, IngestionMode: "background"}
}
