package svc

import (
	"context"
	"time"

	"go-zero-admin/pkg/vectordb"
)

func checkKbVectorReady(ctx context.Context, store vectordb.Store) error {
	probe, ok := store.(interface{ Ready(context.Context) error })
	if !ok {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return probe.Ready(ctx)
}
