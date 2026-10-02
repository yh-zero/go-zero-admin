package accessutil

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/middlecasbin"
	"gorm.io/gorm"
)

// Serialize policy writes inside this RPC process. Business data and policies
// commit in one database transaction; only then queue the enforcement refresh.
var policyWrite = make(chan struct{}, 1)

func PolicyTransaction(ctx context.Context, s *svc.ServiceContext, change func(*gorm.DB) error) error {
	select {
	case policyWrite <- struct{}{}:
	case <-ctx.Done():
		return ctx.Err()
	}
	defer func() { <-policyWrite }()
	if err := s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := LockAdminGuard(tx); err != nil {
			return err
		}
		before, err := adminRecoveryPolicyState(tx)
		if err != nil {
			return err
		}
		if err := change(tx); err != nil {
			return err
		}
		if err := preserveAdminRecoveryPolicies(tx, before); err != nil {
			return err
		}
		if err := middlecasbin.BumpPolicyVersion(tx); err != nil {
			return err
		}
		request := audit.RequestFromContext(ctx)
		return audit.Record(ctx, tx, audit.Event{Module: "permission", Action: "commitPolicy", Object: request.Path})
	}); err != nil {
		return FriendlyDuplicate(err)
	}
	if s.PolicySync != nil {
		// Redis publication and cache reload run on one coalescing worker. Every
		// Enforce still checks the durable version synchronously and fails closed.
		s.PolicySync.Notify()
	} else if s.Casbin != nil {
		if err := s.Casbin.LoadPolicy(); err != nil {
			return err
		}
	}
	return nil
}
