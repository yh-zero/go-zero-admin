package casbinauth

import (
	"sync/atomic"

	"github.com/casbin/casbin/v2"
)

// Enforcer 只暴露请求鉴权所需的方法，调用方不直接改写已发布的策略。
type Enforcer interface {
	Enforce(...interface{}) (bool, error)
}

// PolicyHolder 保持中间件引用稳定，策略实例在构建完成后统一发布。
type PolicyHolder struct {
	current atomic.Pointer[casbin.CachedEnforcer]
}

func NewPolicyHolder(rules []PolicyRule) (*PolicyHolder, error) {
	enforcer, err := NewEnforcer(rules)
	if err != nil {
		return nil, err
	}
	holder := &PolicyHolder{}
	holder.current.Store(enforcer)
	return holder, nil
}

func (h *PolicyHolder) Enforce(values ...interface{}) (bool, error) {
	return h.current.Load().Enforce(values...)
}

func (h *PolicyHolder) Refresh(rules []PolicyRule) error {
	enforcer, err := NewEnforcer(rules)
	if err != nil {
		return err
	}
	h.current.Store(enforcer)
	return nil
}
