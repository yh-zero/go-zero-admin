package audit

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAuditLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAuditLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuditLogListLogic {
	return &GetAuditLogListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAuditLogListLogic) GetAuditLogList(req *types.GetAuditLogListReq) (resp *types.GetAuditLogListResp, err error) {
	rpcResp, err := l.svcCtx.AuditRpc.GetAuditLogList(l.ctx, &adminpb.GetAuditLogListRequest{
		Page:   &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Module: req.Module, Action: req.Action, UserId: req.UserId, Result: req.Result,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AuditLogInfo, 0, len(rpcResp.List))
	for _, a := range rpcResp.List {
		list = append(list, &types.AuditLogInfo{
			Id: a.Id, UserId: a.UserId, Username: a.Username, Module: a.Module, Action: a.Action,
			Method: a.Method, Path: a.Path, Result: a.Result, ErrorMsg: a.ErrorMsg,
			Ip: a.Ip, DurationMs: a.DurationMs, CreatedAt: a.CreatedAt,
		})
	}
	return &types.GetAuditLogListResp{List: list, Total: rpcResp.Page.Total}, nil
}
