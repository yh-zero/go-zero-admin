package svc

import (
	"context"

	"go-zero-admin/pkg/agenttools"
)

// newKbAnalyzer 复用当前服务的检索器与模型；先查 MySQL 归属，再允许检索器请求 Embedding。
func newKbAnalyzer(s *ServiceContext) agenttools.Tool {
	if s.KbSearcher == nil || s.LLM == nil || s.AiKbModel == nil {
		return nil
	}
	return agenttools.NewKbAnalyze(s.KbSearcher, func(ctx context.Context, kbID, ownerID int64) error {
		_, err := s.AiKbModel.FindOwned(ctx, kbID, ownerID)
		return err
	}, s.LLM, s.Config.AI.MaxContextTokens)
}
