package svc

import (
	"context"
	"errors"
	"sort"

	"go-zero-admin/application/ai/rpc/internal/model"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/llm"
	"go-zero-admin/pkg/vectordb"
)

// kbSearcher 知识库检索器（agenttools.KbSearcher 实现）。
// 流程：embed(query) → 向量索引召回（owner/kb 过滤）→ MySQL 复核 status/归属/软删 → 预算截断。
type kbSearcher struct {
	store         vectordb.Store
	embedder      *llm.Embedder
	chunks        model.AiKbChunkModel
	snippetBudget int64 // 检索片段总 token 上限（MaxContextTokens 的 25%）
}

const (
	kbSearchRecallTopK = 20 // 召回数量
	kbSearchReturnTopK = 5  // 返回给模型的片段数
)

func (s *kbSearcher) Search(ctx context.Context, ownerID, kbID int64, query string) ([]agenttools.KbSearchHit, error) {
	vecs, err := s.embedder.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	if len(vecs) != 1 {
		return nil, errors.New("embedding 返回数量异常")
	}
	hits, err := s.store.Search(ctx, vecs[0], ownerID, kbID, kbSearchRecallTopK)
	if err != nil {
		return nil, err
	}
	if len(hits) == 0 {
		return nil, nil
	}
	ids := make([]uint64, 0, len(hits))
	scoreByID := make(map[uint64]float32, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
		scoreByID[h.ID] = h.Score
	}
	// MySQL 复核：向量索引是影子，可能滞后；status/归属/软删以库为准
	rows, err := s.chunks.FindReadyByIDs(ctx, ids, ownerID)
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return scoreByID[uint64(rows[i].Id)] > scoreByID[uint64(rows[j].Id)] })

	out := make([]agenttools.KbSearchHit, 0, kbSearchReturnTopK)
	var budget int64
	for _, r := range rows {
		if len(out) >= kbSearchReturnTopK {
			break
		}
		if budget+r.TokenCount > s.snippetBudget {
			break // 共享上下文预算：检索片段不挤占对话历史
		}
		budget += r.TokenCount
		out = append(out, agenttools.KbSearchHit{
			KbID:     r.KbId,
			DocID:    r.DocId,
			DocTitle: r.DocTitle,
			Snippet:  r.Content,
			Score:    scoreByID[uint64(r.Id)],
		})
	}
	return out, nil
}
