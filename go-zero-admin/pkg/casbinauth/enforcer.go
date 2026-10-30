// Package casbinauth Casbin enforcer 构建（domains 模型，dom=部门）
package casbinauth

import (
	"fmt"
	"time"

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

// NewEnforcer 用给定规则创建 CachedEnforcer
func NewEnforcer(rules []PolicyRule) (*casbin.CachedEnforcer, error) {
	m, err := model.NewModelFromString(ModelText)
	if err != nil {
		return nil, err
	}
	adapter := &sliceAdapter{rules: rules}
	enforcer, err := casbin.NewCachedEnforcer(m, adapter)
	if err != nil {
		return nil, err
	}
	enforcer.SetExpireTime(300 * time.Second)
	return enforcer, nil
}
