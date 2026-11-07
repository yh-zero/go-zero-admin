package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestEmbedQwenBatchLimitAndOrder(t *testing.T) {
	for _, model := range []string{"text-embedding-v3", "text-embedding-v4"} {
		for _, textCount := range []int{11, 26} {
			t.Run(fmt.Sprintf("%s/%d", model, textCount), func(t *testing.T) {
				var requestCount int
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					var req embedRequest
					if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
						t.Errorf("decode request: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					requestCount++
					if req.Model != model {
						t.Errorf("model = %q, want %q", req.Model, model)
					}
					if len(req.Input) > 10 {
						t.Errorf("batch contains %d texts, limit is 10", len(req.Input))
					}
					data := make([]map[string]any, 0, len(req.Input))
					for i := len(req.Input) - 1; i >= 0; i-- {
						index, err := strconv.Atoi(strings.TrimPrefix(req.Input[i], "text-"))
						if err != nil {
							t.Errorf("parse input %q: %v", req.Input[i], err)
							continue
						}
						data = append(data, map[string]any{"index": i, "embedding": []float32{float32(index)}})
					}
					_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
				}))
				defer server.Close()

				embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "test", Model: model, Dims: 1})
				if err != nil {
					t.Fatal(err)
				}
				texts := make([]string, textCount)
				for i := range texts {
					texts[i] = fmt.Sprintf("text-%02d", i)
				}

				vectors, err := embedder.Embed(context.Background(), texts)
				if err != nil {
					t.Fatalf("Embed() error = %v", err)
				}
				wantRequests := (textCount + 9) / 10
				if requestCount != wantRequests {
					t.Errorf("request count = %d, want %d", requestCount, wantRequests)
				}
				if len(vectors) != textCount {
					t.Fatalf("vector count = %d, want %d", len(vectors), textCount)
				}
				for i, vector := range vectors {
					if len(vector) != 1 || vector[0] != float32(i) {
						t.Errorf("vectors[%d] = %v, want [%d]", i, vector, i)
					}
				}
			})
		}
	}
}

func TestEmbedBatchFailureReturnsNoPartialVectors(t *testing.T) {
	var requestCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": embedTestData(10)})
	}))
	defer server.Close()

	embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "test", Model: "text-embedding-v4", Dims: 1})
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := embedder.Embed(context.Background(), make([]string, 11))
	if err == nil {
		t.Fatal("Embed() error = nil, want failure from second batch")
	}
	if vectors != nil {
		t.Errorf("Embed() vectors = %v, want nil on partial failure", vectors)
	}
	if requestCount != 2 {
		t.Errorf("request count = %d, want 2", requestCount)
	}
}

func TestEmbedCancellationDoesNotStartNextBatch(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		cancel()
		_ = json.NewEncoder(w).Encode(map[string]any{"data": embedTestData(10)})
	}))
	defer server.Close()

	embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "test", Model: "text-embedding-v4", Dims: 1})
	if err != nil {
		t.Fatal(err)
	}
	vectors, err := embedder.Embed(ctx, make([]string, 11))
	if err == nil {
		t.Fatal("Embed() error = nil, want canceled context error")
	}
	if vectors != nil {
		t.Errorf("Embed() vectors = %v, want nil on cancellation", vectors)
	}
	if got := requestCount.Load(); got != 1 {
		t.Errorf("request count = %d, want 1", got)
	}
}

func TestEmbedRejectsDuplicateAndMissingIndexes(t *testing.T) {
	for _, test := range []struct {
		name string
		data []map[string]any
	}{
		{name: "duplicate index", data: []map[string]any{{"index": 0, "embedding": []int{1}}, {"index": 0, "embedding": []int{2}}}},
		{name: "missing index", data: []map[string]any{{"index": 0, "embedding": []int{1}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(map[string]any{"data": test.data})
			}))
			defer server.Close()

			embedder, err := NewEmbedder(EmbedConfig{BaseURL: server.URL, APIKey: "test", Model: "text-embedding-v4", Dims: 1})
			if err != nil {
				t.Fatal(err)
			}
			vectors, err := embedder.Embed(context.Background(), []string{"first", "second"})
			if err == nil {
				t.Fatal("Embed() error = nil, want malformed index response error")
			}
			if vectors != nil {
				t.Errorf("Embed() vectors = %v, want nil on malformed response", vectors)
			}
		})
	}
}

func embedTestData(count int) []map[string]any {
	data := make([]map[string]any, count)
	for i := range data {
		data[i] = map[string]any{"index": i, "embedding": []int{i}}
	}
	return data
}
