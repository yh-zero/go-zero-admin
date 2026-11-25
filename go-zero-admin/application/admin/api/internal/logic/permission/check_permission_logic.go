package permission

import (
	"context"
	"fmt"
	"strings"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckPermissionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCheckPermissionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckPermissionLogic {
	return &CheckPermissionLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CheckPermissionLogic) CheckPermission(req *types.PermissionCheckReq) (*types.PermissionCheckResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor := middleware.ActorFromContext(l.ctx)
	if actor.GetUserId() <= 0 || !strings.HasPrefix(req.Path, "/v1/") {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	current, err := l.svcCtx.UserRpc.GetUser(l.ctx, &adminpb.GetUserRequest{Actor: actor, UserId: actor.UserId})
	if err != nil {
		return nil, err
	}
	if current.User == nil || !current.User.IsSuperAdmin {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	userID := req.UserId
	if userID == 0 {
		userID = actor.UserId
	}
	deptID := actor.DeptId
	if req.DeptId != nil {
		deptID = *req.DeptId
	}
	allowed, role, domain, err := l.svcCtx.Enforcer.Explain(
		fmt.Sprintf("user:%d", userID), fmt.Sprintf("dept:%d", deptID), req.Path, req.Method,
	)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	revision, generatedAt, loadedAt := l.svcCtx.PolicySnapshot()
	resp := &types.PermissionCheckResp{
		Allowed: allowed, Role: role, Domain: domain,
		PolicyRevision: revision, GeneratedAt: generatedAt, LoadedAt: loadedAt,
		DataScopes: []string{},
	}
	if allowed && req.Path == "/v1/sys/user/list" && req.Method == "GET" {
		snapshot, err := l.svcCtx.UserRpc.GetUserPermissions(l.ctx, &adminpb.GetUserPermissionsRequest{
			Actor: actor, UserId: userID, DeptId: &deptID,
		})
		if err != nil {
			return nil, err
		}
		resp.DataScopes = append(resp.DataScopes, snapshot.DataScopes...)
	}
	return resp, nil
}
