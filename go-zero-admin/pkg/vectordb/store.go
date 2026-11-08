// Package vectordb 知识库向量索引封装。
// 纪律：MySQL 是唯一事实源（chunk 正文/状态/归属）；本包只操作派生向量索引，
// 所有写入必须幂等可重放，任何时刻整体数据可从 MySQL 重建。
package vectordb

import (
	"context"
	"errors"
)

// ErrReconcileLimit 表示完整索引快照超过扫描上限，必须人工评估后处理。
var ErrReconcileLimit = errors.New("vector reconciliation snapshot limit exceeded")

// Point 一条向量索引记录；ID 即 ai_kb_chunk.id
type Point struct {
	ID         uint64
	Vector     []float32
	KbID       int64
	OwnerID    int64
	DocID      int64
	TokenCount int64
}

// Hit 一次召回结果；正文必须回 MySQL 复核后取用
type Hit struct {
	ID    uint64
	Score float32
}

// Store 向量索引操作集合；实现必须保证幂等（Upsert/Delete 可重复调用）
type Store interface {
	// EnsureCollection 按当前 embedding 模型与维度确保集合存在（含过滤字段索引）
	EnsureCollection(ctx context.Context) error
	// Upsert 批量写入/覆盖向量点
	Upsert(ctx context.Context, points []Point) error
	// Search 按 owner（必选）与 kb（0 为全部）过滤检索 topK
	Search(ctx context.Context, vector []float32, ownerID, kbID int64, topK int) ([]Hit, error)
	// Delete 按 point ID 删除（不存在的 ID 不视为错误）
	Delete(ctx context.Context, ids []uint64) error
	// DeleteByKb 按 kb_id 过滤删除（知识库级联删除用）
	DeleteByKb(ctx context.Context, kbID int64) error
	// AllIDs 列出全部 point ID（对账用；数据量超预算时返回错误而非截断）
	AllIDs(ctx context.Context) (map[uint64]struct{}, error)
	// Close 释放连接
	Close() error
}
