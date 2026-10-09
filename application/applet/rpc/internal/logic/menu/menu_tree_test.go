package menulogic

import (
	"fmt"
	"reflect"
	"testing"

	"go-zero-admin/application/applet/rpc/internal/model"
	base "go-zero-admin/pkg/model"
)

// This is the original repeated-scan algorithm retained only as a benchmark baseline.
func scanBaseMenuTree(all []model.SysBaseMenu, parent int64) []model.SysBaseMenu {
	result := make([]model.SysBaseMenu, 0)
	for _, menu := range all {
		if menu.ParentId == parent {
			menu.Children = scanBaseMenuTree(all, menu.ID)
			result = append(result, menu)
		}
	}
	return result
}

func TestBaseMenuTreePreservesOrderPayloadAndEmptyChildren(t *testing.T) {
	all := []model.SysBaseMenu{
		{MODEL_BASE: base.MODEL_BASE{ID: 2}, Name: "root-first", Sort: 1},
		{MODEL_BASE: base.MODEL_BASE{ID: 4}, ParentId: 2, Name: "child-first", Sort: 1, Parameters: []model.SysBaseMenuParameter{{Key: "key", Value: "value"}}},
		{MODEL_BASE: base.MODEL_BASE{ID: 1}, Name: "root-second", Sort: 2},
		{MODEL_BASE: base.MODEL_BASE{ID: 3}, ParentId: 2, Name: "child-second", Sort: 2},
		{MODEL_BASE: base.MODEL_BASE{ID: 5}, ParentId: 99, Name: "orphan"},
	}
	before := append([]model.SysBaseMenu(nil), all...)
	got := baseMenuTree(all, 0)
	if len(got) != 2 || got[0].ID != 2 || got[1].ID != 1 || len(got[0].Children) != 2 || got[0].Children[0].ID != 4 || got[0].Children[1].ID != 3 {
		t.Fatalf("tree order or orphan semantics changed: %+v", got)
	}
	if got[0].Children[0].Parameters[0].Value != "value" || got[1].Children == nil {
		t.Fatal("payload or empty array semantics changed")
	}
	if !reflect.DeepEqual(before, all) {
		t.Fatal("tree construction modified input")
	}
	if children := baseMenuTree(all, 2); len(children) != 2 || children[0].ID != 4 {
		t.Fatal("non-root tree entry changed")
	}
}

var menuTreeBenchmarkResult []model.SysBaseMenu

func BenchmarkBaseMenuTree(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		all := make([]model.SysBaseMenu, n)
		for i := range all {
			all[i].ID = int64(i + 1)
			all[i].ParentId = int64(i / 8)
		}
		for _, candidate := range []struct {
			name  string
			build func([]model.SysBaseMenu, int64) []model.SysBaseMenu
		}{{"scan", scanBaseMenuTree}, {"current", baseMenuTree}} {
			b.Run(fmt.Sprintf("%s/%d", candidate.name, n), func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					menuTreeBenchmarkResult = candidate.build(all, 0)
				}
			})
		}
	}
}
