// Package casbinauth Casbin enforcer 构建（domains 模型，dom=部门）
package casbinauth

import (
	"fmt"
	"sync/atomic"

	"github.com/casbin/casbin/v2"
	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
)

// ModelText Casbin domains 模型（dom = 部门）
const ModelText = `
[request_definition]
r = sub, dom, obj, act

[policy_definition]
p = sub, dom, obj, act

[role_definition]
g = _, _, _

[policy_effect]
e = some(where (p.eft == allow))

[matchers]
m = (g(r.sub, p.sub, r.dom) || g(r.sub, p.sub, "*")) && (r.dom == p.dom || p.dom == "*") && keyMatch2(r.obj, p.obj) && (r.act == p.act || p.act == "*")
`

// PolicyRule 策略规则（与 admin.proto 的 PolicyRule 对齐，pkg 不依赖 internal 包）
type PolicyRule struct {
	Ptype string
	V0    string
	V1    string
	V2    string
	V3    string
	V4    string
	V5    string
}

// sliceAdapter 内存策略 adapter
type sliceAdapter struct {
	rules []PolicyRule
}

func (a *sliceAdapter) LoadPolicy(m model.Model) error {
	for _, r := range a.rules {
		line := r.Ptype
		for _, v := range []string{r.V0, r.V1, r.V2, r.V3, r.V4, r.V5} {
			if v != "" {
				line += ", " + v
			}
		}
		persist.LoadPolicyLine(line, m)
	}
	return nil
}

func (a *sliceAdapter) SavePolicy(model.Model) error { return fmt.Errorf("not implemented") }
func (a *sliceAdapter) AddPolicy(string, string, []string) error {
	return fmt.Errorf("not implemented")
}
func (a *sliceAdapter) RemovePolicy(string, string, []string) error {
	return fmt.Errorf("not implemented")
}
func (a *sliceAdapter) RemoveFilteredPolicy(string, string, int, ...string) error {
	return fmt.Errorf("not implemented")
}

// Holder 并发安全持有当前 enforcer。
// 刷新语义为全量构建+原子替换，读路径无锁；
// 不用 casbin.CachedEnforcer：其缓存带过期时间的并发实现出过事故，
// 且整体替换语义下实例内缓存没有意义。
type Holder struct {
	ptr atomic.Pointer[casbin.Enforcer]
}

// NewHolder 以初始规则创建 Holder
func NewHolder(rules []PolicyRule) (*Holder, error) {
	e, err := buildEnforcer(rules)
	if err != nil {
		return nil, err
	}
	h := &Holder{}
	h.ptr.Store(e)
	return h, nil
}

// Enforce 对当前实例执行鉴权（纯内存，微秒级）
func (h *Holder) Enforce(sub, dom, obj, act string) (bool, error) {
	return h.ptr.Load().Enforce(sub, dom, obj, act)
}

// Swap 原子替换 enforcer（策略全量刷新时调用）
func (h *Holder) Swap(rules []PolicyRule) error {
	e, err := buildEnforcer(rules)
	if err != nil {
		return err
	}
	h.ptr.Store(e)
	return nil
}

func buildEnforcer(rules []PolicyRule) (*casbin.Enforcer, error) {
	m, err := model.NewModelFromString(ModelText)
	if err != nil {
		return nil, err
	}
	return casbin.NewEnforcer(m, &sliceAdapter{rules: rules})
}
