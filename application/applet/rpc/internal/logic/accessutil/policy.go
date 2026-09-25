package accessutil

import (
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

// Serialize policy writes inside this RPC process. Business data and policies
// commit in one database transaction; only then reload the enforcement cache.
var policyWrite sync.Mutex

func PolicyTransaction(s *svc.ServiceContext, change func(*gorm.DB) error) error {
	policyWrite.Lock()
	defer policyWrite.Unlock()
	if err := s.DB.Transaction(func(tx *gorm.DB) error {
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
		return preserveAdminRecoveryPolicies(tx, before)
	}); err != nil {
		return FriendlyDuplicate(err)
	}
	if s.Casbin != nil {
		if err := s.Casbin.LoadPolicy(); err != nil {
			logx.Errorf("Reload committed policies: %v", err)
			return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "数据已保存，但权限缓存刷新失败，请刷新确认后重试")
		}
	}
	if s.BizRedis != nil {
		// Same message used by the existing redis-watcher for a complete policy reload.
		if _, err := s.BizRedis.Publish("/casbin", `{"Method":"Update","ID":"business-policy-transaction"}`); err != nil {
			logx.Errorf("Broadcast committed policies: %v", err)
			return xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "数据已保存，但权限同步失败，请刷新确认后重试")
		}
	}
	return nil
}
