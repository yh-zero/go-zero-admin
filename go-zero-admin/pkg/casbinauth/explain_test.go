package casbinauth

import "testing"

func TestExplainReturnsMatchedPolicyForCurrentDepartment(t *testing.T) {
	holder, err := NewHolder([]PolicyRule{
		{Ptype: "g", V0: "user:7", V1: "dept_reader", V2: "dept:3"},
		{Ptype: "p", V0: "dept_reader", V1: "*", V2: "/v1/sys/user/:id", V3: "GET"},
	})
	if err != nil {
		t.Fatal(err)
	}
	allowed, role, domain, err := holder.Explain("user:7", "dept:3", "/v1/sys/user/7", "GET")
	if err != nil || !allowed || role != "dept_reader" || domain != "*" {
		t.Fatalf("allowed=%v role=%q domain=%q err=%v", allowed, role, domain, err)
	}
	allowed, role, domain, err = holder.Explain("user:7", "dept:4", "/v1/sys/user/7", "GET")
	if err != nil || allowed || role != "" || domain != "" {
		t.Fatalf("allowed=%v role=%q domain=%q err=%v", allowed, role, domain, err)
	}
}
