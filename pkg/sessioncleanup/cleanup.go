package sessioncleanup

import (
	"context"
	"errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Options struct {
	RetentionDays, BatchSize int
	Apply                    bool
	Now                      time.Time
}
type Result struct {
	DryRun   bool      `json:"dryRun"`
	Cutoff   time.Time `json:"cutoff"`
	Eligible int64     `json:"eligible"`
	Deleted  int64     `json:"deleted"`
	Batches  int64     `json:"batches"`
}
type Session struct {
	ID        string `gorm:"primaryKey"`
	CreatedAt time.Time
	ExpiresAt time.Time
	RevokedAt *time.Time
}

func (Session) TableName() string { return "sys_device_sessions" }

// Run only deletes old terminal rows. A recent revocation starts a fresh retention
// interval even if the session had expired before it was revoked.
func Run(ctx context.Context, db *gorm.DB, options Options) (Result, error) {
	result := Result{DryRun: !options.Apply}
	if options.RetentionDays < 1 || options.RetentionDays > 36500 || options.BatchSize < 1 || options.BatchSize > 5000 {
		return result, errors.New("retention-days (1..36500) and batch-size (1..5000) must be explicitly set")
	}
	if db == nil {
		return result, errors.New("session database is required")
	}
	now := options.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	result.Cutoff = now.AddDate(0, 0, -options.RetentionDays)
	eligible := func(query *gorm.DB) *gorm.DB {
		return query.Where("created_at < ? AND ((revoked_at IS NULL AND expires_at < ?) OR (revoked_at IS NOT NULL AND revoked_at < ?))", result.Cutoff, result.Cutoff, result.Cutoff)
	}
	if err := eligible(db.WithContext(ctx).Model(&Session{})).Count(&result.Eligible).Error; err != nil {
		return result, err
	}
	if result.DryRun {
		return result, nil
	}
	for {
		var deleted int64
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var rows []Session
			if err := eligible(tx).Select("id").Order("expires_at,id").Limit(options.BatchSize).Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).Find(&rows).Error; err != nil {
				return err
			}
			if len(rows) == 0 {
				return nil
			}
			ids := make([]string, len(rows))
			for i := range rows {
				ids[i] = rows[i].ID
			}
			operation := eligible(tx).Where("id IN ?", ids).Delete(&Session{})
			deleted = operation.RowsAffected
			return operation.Error
		})
		if err != nil {
			return result, err
		}
		if deleted == 0 {
			break
		}
		result.Deleted += deleted
		result.Batches++
	}
	return result, nil
}
