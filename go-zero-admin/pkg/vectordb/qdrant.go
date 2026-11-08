package vectordb

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

// QdrantStore Qdrant 实现的向量索引。
// 集合名携带 embedding 模型与维度：更换模型/维度 = 新集合 + 全量重嵌入，不做原地迁移。
type QdrantStore struct {
	client     *qdrant.Client
	collection string
	dims       uint64
}

var collectionNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_]+`)

// CollectionName 生成集合名，如 kb_chunks_qwen_text-embedding-v4_1024
func CollectionName(provider, model string, dims int64) string {
	sanitized := collectionNameSanitizer.ReplaceAllString(strings.ToLower(strings.TrimSpace(model)), "_")
	return fmt.Sprintf("kb_chunks_%s_%s_%d", strings.ToLower(strings.TrimSpace(provider)), sanitized, dims)
}

// NewQdrantStore 创建 Qdrant 向量索引；address 形如 qdrant:6334 或 127.0.0.1:16334
func NewQdrantStore(address, apiKey, collection string, dims int64) (*QdrantStore, error) {
	host, port, err := splitAddress(address)
	if err != nil {
		return nil, err
	}
	client, err := qdrant.NewClient(&qdrant.Config{
		Host:   host,
		Port:   port,
		APIKey: strings.TrimSpace(apiKey),
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant 连接失败: %w", err)
	}
	return &QdrantStore{client: client, collection: collection, dims: uint64(dims)}, nil
}

func splitAddress(address string) (string, int, error) {
	host, portStr, ok := strings.Cut(strings.TrimSpace(address), ":")
	if !ok || host == "" {
		return "", 0, fmt.Errorf("qdrant 地址无效: %q", address)
	}
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil || port <= 0 || port > 65535 {
		return "", 0, fmt.Errorf("qdrant 端口无效: %q", portStr)
	}
	return host, port, nil
}

func (s *QdrantStore) Close() error { return s.client.Close() }

func ptrBool(v bool) *bool { return &v }

// EnsureCollection 集合不存在时创建（Cosine 距离），并确保 owner_id/kb_id 过滤索引存在
func (s *QdrantStore) EnsureCollection(ctx context.Context) error {
	exists, err := s.client.CollectionExists(ctx, s.collection)
	if err != nil {
		return fmt.Errorf("qdrant 集合检查失败: %w", err)
	}
	if !exists {
		if err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: s.collection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     s.dims,
				Distance: qdrant.Distance_Cosine,
			}),
		}); err != nil {
			return fmt.Errorf("qdrant 建集合失败: %w", err)
		}
	}
	// payload 过滤索引（幂等：已存在时服务端忽略或报已存在，按错误信息容忍）
	intType := qdrant.FieldType_FieldTypeInteger
	for _, field := range []string{"owner_id", "kb_id"} {
		_, err := s.client.CreateFieldIndex(ctx, &qdrant.CreateFieldIndexCollection{
			CollectionName: s.collection,
			FieldName:      field,
			FieldType:      &intType,
		})
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("qdrant 建字段索引 %s 失败: %w", field, err)
		}
	}
	return nil
}

func (s *QdrantStore) Upsert(ctx context.Context, points []Point) error {
	if len(points) == 0 {
		return nil
	}
	structs := make([]*qdrant.PointStruct, 0, len(points))
	for _, p := range points {
		if len(p.Vector) != int(s.dims) {
			return fmt.Errorf("point %d 向量维度 %d 与集合 %d 不一致", p.ID, len(p.Vector), s.dims)
		}
		structs = append(structs, &qdrant.PointStruct{
			Id:      qdrant.NewIDNum(p.ID),
			Vectors: qdrant.NewVectors(p.Vector...),
			Payload: qdrant.NewValueMap(map[string]any{
				"kb_id":       p.KbID,
				"owner_id":    p.OwnerID,
				"doc_id":      p.DocID,
				"token_count": p.TokenCount,
			}),
		})
	}
	// wait=true：写入生效后才返回，摄取终态与检索建立在已提交的索引上
	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: s.collection,
		Points:         structs,
		Wait:           ptrBool(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant upsert 失败: %w", err)
	}
	return nil
}

func (s *QdrantStore) Search(ctx context.Context, vector []float32, ownerID, kbID int64, topK int) ([]Hit, error) {
	if len(vector) != int(s.dims) {
		return nil, fmt.Errorf("查询向量维度 %d 与集合 %d 不一致", len(vector), s.dims)
	}
	must := []*qdrant.Condition{qdrant.NewMatchInt("owner_id", ownerID)}
	if kbID > 0 {
		must = append(must, qdrant.NewMatchInt("kb_id", kbID))
	}
	limit := uint64(topK)
	scored, err := s.client.Query(ctx, &qdrant.QueryPoints{
		CollectionName: s.collection,
		Query:          qdrant.NewQuery(vector...),
		Filter:         &qdrant.Filter{Must: must},
		Limit:          &limit,
	})
	if err != nil {
		return nil, fmt.Errorf("qdrant 检索失败: %w", err)
	}
	hits := make([]Hit, 0, len(scored))
	for _, sp := range scored {
		if num := sp.GetId().GetNum(); num > 0 {
			hits = append(hits, Hit{ID: num, Score: sp.GetScore()})
		}
	}
	return hits, nil
}

func (s *QdrantStore) Delete(ctx context.Context, ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	pointIDs := make([]*qdrant.PointId, 0, len(ids))
	for _, id := range ids {
		pointIDs = append(pointIDs, qdrant.NewIDNum(id))
	}
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Points:         qdrant.NewPointsSelectorIDs(pointIDs),
		Wait:           ptrBool(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant 删除失败: %w", err)
	}
	return nil
}

func (s *QdrantStore) DeleteByKb(ctx context.Context, kbID int64) error {
	_, err := s.client.Delete(ctx, &qdrant.DeletePoints{
		CollectionName: s.collection,
		Points: qdrant.NewPointsSelectorFilter(&qdrant.Filter{
			Must: []*qdrant.Condition{qdrant.NewMatchInt("kb_id", kbID)},
		}),
		Wait: ptrBool(true),
	})
	if err != nil {
		return fmt.Errorf("qdrant 按知识库删除失败: %w", err)
	}
	return nil
}

// AllIDs 全量列出 point ID（对账用）；超过上限返回错误，不允许截断后冒充全量
func (s *QdrantStore) AllIDs(ctx context.Context) (map[uint64]struct{}, error) {
	const maxPoints = 1_000_000
	const pageSize = 1024
	out := make(map[uint64]struct{})
	var offset *qdrant.PointId
	for {
		limit := uint32(pageSize)
		points, err := s.client.Scroll(ctx, &qdrant.ScrollPoints{
			CollectionName: s.collection,
			Offset:         offset,
			Limit:          &limit,
		})
		if err != nil {
			return nil, fmt.Errorf("qdrant scroll 失败: %w", err)
		}
		for _, p := range points {
			if num := p.GetId().GetNum(); num > 0 {
				out[num] = struct{}{}
			}
		}
		if len(out) > maxPoints {
			return nil, fmt.Errorf("%w: qdrant point 数超过对账上限 %d", ErrReconcileLimit, maxPoints)
		}
		if len(points) < pageSize {
			return out, nil
		}
		offset = points[len(points)-1].GetId()
	}
}
