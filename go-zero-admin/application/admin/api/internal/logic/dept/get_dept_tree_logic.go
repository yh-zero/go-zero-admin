package dept

import (
	"context"
	"fmt"
	"go-zero-admin/application/admin/api/internal/middleware"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeptTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDeptTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeptTreeLogic {
	return &GetDeptTreeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetDeptTreeLogic) GetDeptTree(req *types.GetDeptTreeReq) (resp *types.GetDeptTreeResp, err error) {
	actor := middleware.ActorFromContext(l.ctx)
	includeMembers, err := departmentMembersReadable(l.svcCtx, actor)
	if err != nil {
		l.Error(err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	rpcResp, err := l.svcCtx.DeptRpc.GetDeptTree(l.ctx, &adminpb.GetDeptTreeRequest{Actor: actor, Status: req.Status, IncludeMembers: includeMembers})
	if err != nil {
		return nil, err
	}
	list := make([]*types.DeptInfo, 0, len(rpcResp.List))
	for _, d := range rpcResp.List {
		list = append(list, mapDeptInfo(d))
	}
	return &types.GetDeptTreeResp{List: list}, nil
}

func mapDeptInfo(d *adminpb.DeptInfo) *types.DeptInfo {
	return &types.DeptInfo{
		Id: d.Id, ParentId: d.ParentId, Name: d.Name, Leader: d.Leader,
		Phone: d.Phone, Email: d.Email, Sort: d.Sort, Status: d.Status,
		Path: d.Path, CreatedAt: d.CreatedAt,
		LeaderUserId: d.LeaderUserId, MemberCount: d.MemberCount, MembersVisible: d.MembersVisible,
	}
}

func departmentMembersReadable(svcCtx *svc.ServiceContext, actor *adminpb.Actor) (bool, error) {
	return svcCtx.Enforcer.Enforce(fmt.Sprintf("user:%d", actor.GetUserId()), fmt.Sprintf("dept:%d", actor.GetDeptId()), "/v1/sys/user/list", "GET")
}

func checkDepartmentLeaderAccess(svcCtx *svc.ServiceContext, actor *adminpb.Actor, leaderUserId int64) error {
	if leaderUserId <= 0 {
		return nil
	}
	allowed, err := departmentMembersReadable(svcCtx, actor)
	if err != nil {
		return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if !allowed {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	return nil
}
