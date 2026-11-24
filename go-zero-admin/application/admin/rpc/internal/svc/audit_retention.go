package svc

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// StartAuditRetentionWorker 默认不删除审计；显式配置正整数保留天数后按批清理。
func (s *ServiceContext) StartAuditRetentionWorker(ctx context.Context) error {
	days := s.Config.Audit.RetentionDays
	if days == 0 {
		return nil
	}
	interval := s.Config.Audit.CleanupIntervalSeconds
	batch := s.Config.Audit.CleanupBatchSize
	if days < 1 || interval < 60 || batch < 1 || batch > 10000 {
		return fmt.Errorf("invalid Audit retention config")
	}
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()
		for {
			s.purgeExpiredAudit(ctx, days, batch)
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return nil
}

func (s *ServiceContext) purgeExpiredAudit(ctx context.Context, days, batch int) {
	cutoff := time.Now().AddDate(0, 0, -days).Unix()
	deleted, err := s.SysAuditLogModel.PurgeBefore(ctx, cutoff, batch)
	if err != nil {
		logx.Errorf("audit retention cleanup failed: %v", err)
		return
	}
	if deleted > 0 {
		logx.Infof("audit retention cleanup deleted %d rows", deleted)
	}
}
