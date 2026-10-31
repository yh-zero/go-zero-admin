package apiservicelogic

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type SyncApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSyncApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncApiLogic {
	return &SyncApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SyncApiLogic) SyncApi(in *adminpb.SyncApiRequest) (*adminpb.SyncApiResponse, error) {
	allowed, err := l.svcCtx.SysUserModel.IsSuperAdmin(l.ctx, in.GetActor().GetUserId())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if !allowed {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	seen := map[string]bool{}
	routes := make([]model.ApiRouteInput, 0, len(in.Routes))
	for _, route := range in.Routes {
		if route == nil || !strings.HasPrefix(route.Path, "/v1/") || len(route.Path) > 255 || len(route.Group) > 64 || len(route.Title) > 128 {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		switch route.Method {
		case http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		default:
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		key := route.Method + " " + route.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		routes = append(routes, model.ApiRouteInput{Method: route.Method, Path: route.Path, Group: route.Group, Title: route.Title})
	}
	if len(routes) == 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	added, updated, err := l.svcCtx.SysApiModel.SyncRoutes(l.ctx, routes)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.SyncApiResponse{Added: added, Updated: updated, Removed: 0}, nil
}
