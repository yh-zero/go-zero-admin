package casbinservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type ListPoliciesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListPoliciesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListPoliciesLogic {
	return &ListPoliciesLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// ListPolicies 全量返回 casbin_rule（API 层初始化/刷新 enforcer 用）
func (l *ListPoliciesLogic) ListPolicies(in *adminpb.ListPoliciesRequest) (*adminpb.ListPoliciesResponse, error) {
	type row struct {
		Ptype string `db:"ptype"`
		V0    string `db:"v0"`
		V1    string `db:"v1"`
		V2    string `db:"v2"`
		V3    string `db:"v3"`
		V4    string `db:"v4"`
		V5    string `db:"v5"`
	}
	var rows []row
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT ptype, v0, v1, v2, v3, v4, v5 FROM casbin_rule")
	if err != nil {
		l.Errorf("query casbin_rule error: %v", err)
		return nil, err
	}
	rules := make([]*adminpb.PolicyRule, 0, len(rows))
	for _, r := range rows {
		rules = append(rules, &adminpb.PolicyRule{
			Ptype: r.Ptype, V0: r.V0, V1: r.V1, V2: r.V2, V3: r.V3, V4: r.V4, V5: r.V5,
		})
	}
	return &adminpb.ListPoliciesResponse{Rules: rules}, nil
}
