// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"
	"sort"
	"strings"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppMembershipPlansLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAppMembershipPlansLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppMembershipPlansLogic {
	return &ListAppMembershipPlansLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppMembershipPlansLogic) ListAppMembershipPlans() (resp *types.AppMembershipPlansResp, err error) {
	list := make([]types.AppMembershipPlanInfo, 0)
	if l.svcCtx.DictRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	r, err := l.svcCtx.DictRpc.GetDictByCode(l.ctx, &adminpb.GetDictByCodeRequest{Code: "app_membership_plan"})
	if err != nil {
		if code, _, found := xerr.FromGRPCError(err); found && code == xerr.DICT_NOT_FOUND {
			return &types.AppMembershipPlansResp{List: list}, nil
		}
		if codeError, found := xerr.AsCodeError(err); found && codeError.GetErrCode() == xerr.DICT_NOT_FOUND {
			return &types.AppMembershipPlansResp{List: list}, nil
		}
		return nil, err
	}
	if r.GetDict().GetStatus() != 1 {
		return &types.AppMembershipPlansResp{List: list}, nil
	}
	items := append([]*adminpb.DictItem(nil), r.GetDict().GetItems()...)
	sort.SliceStable(items, func(i, j int) bool { return items[i].GetSort() < items[j].GetSort() })
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		label, value := strings.TrimSpace(item.GetLabel()), strings.TrimSpace(item.GetValue())
		if label == "" || value == "" || len(value) > 32 || seen[value] {
			continue
		}
		seen[value] = true
		list = append(list, types.AppMembershipPlanInfo{Label: label, Value: value})
	}
	return &types.AppMembershipPlansResp{List: list}, nil
}
