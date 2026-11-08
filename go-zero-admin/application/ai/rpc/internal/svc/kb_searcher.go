package svc

import (
	"context"
	"errors"
	"sort"
	"unicode/utf8"

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
	return s.search(ctx, ownerID, kbID, query, kbSearchReturnTopK)
}

// SearchAnalysis 让单次分析在原召回内选择依据，不提前丢弃低排名的关键片段。
func (s *kbSearcher) SearchAnalysis(ctx context.Context, ownerID, kbID int64, query string) ([]agenttools.KbSearchHit, error) {
	return s.search(ctx, ownerID, kbID, query, kbSearchRecallTopK)
}

func (s *kbSearcher) search(ctx context.Context, ownerID, kbID int64, query string, returnTopK int) ([]agenttools.KbSearchHit, error) {
	remaining := agenttools.KbSearchRemainingTokens(ctx, s.snippetBudget)
	if remaining <= 0 {
		return nil, nil
	}
	if err := checkKbVectorReady(ctx, s.store); err != nil {
		return nil, err
	}
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
	rows, err := s.chunks.FindReadyByIDs(ctx, ids, ownerID, kbID)
	if err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		left, right := scoreByID[uint64(rows[i].Id)], scoreByID[uint64(rows[j].Id)]
		if left == right {
			return rows[i].Id < rows[j].Id
		}
		return left > right
	})
	if returnTopK == kbSearchReturnTopK {
		rows = selectSearchDocuments(rows)
	}

	out := make([]agenttools.KbSearchHit, 0, returnTopK)
	var budget int64
	for _, r := range rows {
		if len(out) >= returnTopK {
			break
		}
		// 不能依赖历史 token_count 低估正文；大块允许取片段，最终 JSON 开销由工具共享额度约束。
		cost := max(r.TokenCount, (int64(len(r.Content))+2)/3)
		content := r.Content
		if budget+cost > remaining {
			if returnTopK == kbSearchRecallTopK {
				// 分析引用必须对应真实正文；JSON资料也只取字节前缀，不重写为另一JSON。
				limit := min(len(content), int((remaining-budget)*3))
				for limit > 0 && limit < len(content) && !utf8.RuneStart(content[limit]) {
					limit--
				}
				content = content[:limit]
			} else {
				content = agenttools.CompactToolResult(content, int((remaining-budget)*3))
			}
			cost = (int64(len(content)) + 2) / 3
		}
		if content == "" {
			break
		}
		budget += cost
		out = append(out, agenttools.KbSearchHit{
			ChunkID:  r.Id,
			KbID:     r.KbId,
			DocID:    r.DocId,
			DocTitle: r.DocTitle,
			Snippet:  content,
			Score:    scoreByID[uint64(r.Id)],
		})
	}
	return out, nil
}

// 集中候选保留主文档上下文；分散候选增加文档覆盖，输出始终保持原排名。
func selectSearchDocuments(ranked []*model.KbChunkRow) []*model.KbChunkRow {
	if len(ranked) <= kbSearchReturnTopK {
		return ranked
	}
	leadingDocuments := make(map[[2]int64]int, kbSearchReturnTopK)
	for _, row := range ranked[:kbSearchReturnTopK] {
		document := [2]int64{row.KbId, row.DocId}
		leadingDocuments[document]++
		if leadingDocuments[document] > kbSearchReturnTopK/2 {
			return ranked[:kbSearchReturnTopK]
		}
	}
	selected := make(map[int64]struct{}, kbSearchReturnTopK)
	documents := make(map[[2]int64]struct{}, kbSearchReturnTopK)
	for _, row := range ranked {
		document := [2]int64{row.KbId, row.DocId}
		if _, exists := documents[document]; exists {
			continue
		}
		documents[document] = struct{}{}
		selected[row.Id] = struct{}{}
		if len(selected) == kbSearchReturnTopK {
			break
		}
	}
	for _, row := range ranked {
		if len(selected) == kbSearchReturnTopK {
			break
		}
		selected[row.Id] = struct{}{}
	}
	rows := make([]*model.KbChunkRow, 0, len(selected))
	for _, row := range ranked {
		if _, exists := selected[row.Id]; exists {
			rows = append(rows, row)
		}
	}
	return rows
}
