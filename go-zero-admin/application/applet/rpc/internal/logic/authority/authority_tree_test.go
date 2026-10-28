package authoritylogic

import (
	"fmt"
	"reflect"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
)

func scanAuthorityTree(all []model.SysAuthority, parent int64) []model.SysAuthority {
	result := make([]model.SysAuthority, 0)
	for _, role := range all {
		p := int64(0)
		if role.ParentId != nil {
			p = *role.ParentId
		}
		if p == parent {
			role.Children = scanAuthorityTree(all, role.AuthorityId)
			result = append(result, role)
		}
	}
	return result
}

func TestAuthorityTreePreservesNilRootOrderAndPayload(t *testing.T) {
	parent, orphan := int64(2), int64(99)
	all := []model.SysAuthority{{AuthorityId: 2, AuthorityName: "first"}, {AuthorityId: 4, ParentId: &parent, AuthorityName: "first-child", DefaultRouter: "home"}, {AuthorityId: 1, AuthorityName: "second"}, {AuthorityId: 3, ParentId: &parent, AuthorityName: "second-child"}, {AuthorityId: 5, ParentId: &orphan}}
	before := append([]model.SysAuthority(nil), all...)
	got := authorityTree(all, 0)
	if len(got) != 2 || got[0].AuthorityId != 2 || got[1].AuthorityId != 1 || len(got[0].Children) != 2 || got[0].Children[0].AuthorityId != 4 || got[0].Children[1].AuthorityId != 3 {
		t.Fatalf("tree order or orphan semantics changed: %+v", got)
	}
	if got[0].Children[0].DefaultRouter != "home" || got[1].Children == nil {
		t.Fatal("role payload or empty array semantics changed")
	}
	if !reflect.DeepEqual(before, all) {
		t.Fatal("tree construction modified input")
	}
	if children := authorityTree(all, 2); len(children) != 2 || children[0].AuthorityId != 4 {
		t.Fatal("non-root tree entry changed")
	}
}

var authorityTreeBenchmarkResult []model.SysAuthority

func BenchmarkAuthorityTree(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		all := make([]model.SysAuthority, n)
		for i := range all {
			all[i].AuthorityId = int64(i + 1)
			parent := int64(i / 8)
			all[i].ParentId = &parent
		}
		for _, candidate := range []struct {
			name  string
			build func([]model.SysAuthority, int64) []model.SysAuthority
		}{{"scan", scanAuthorityTree}, {"current", authorityTree}} {
			b.Run(fmt.Sprintf("%s/%d", candidate.name, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					authorityTreeBenchmarkResult = candidate.build(all, 0)
				}
			})
		}
	}
}
