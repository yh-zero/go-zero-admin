package model

import (
	"testing"
)

func TestKbIngestRequestHashPreservesOperationAndExactBody(t *testing.T) {
	base := KbEnqueueRequest{KbId: 1, DocId: 2, OwnerId: 3, Operation: "update", ExpectedVersion: 4, Title: "文档", Content: "正文\r\n", EmbeddingModel: "local"}
	hash := kbIngestRequestHash(base)
	for _, change := range []func(*KbEnqueueRequest){
		func(r *KbEnqueueRequest) { r.Operation = "create" },
		func(r *KbEnqueueRequest) { r.Content = "正文\n" },
		func(r *KbEnqueueRequest) { r.ExpectedVersion++ },
		func(r *KbEnqueueRequest) { r.BatchId = "other-batch" },
	} {
		candidate := base
		change(&candidate)
		if kbIngestRequestHash(candidate) == hash {
			t.Fatal("different request reused the same identity")
		}
	}
}
