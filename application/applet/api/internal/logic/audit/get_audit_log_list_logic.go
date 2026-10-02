package audit

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

type GetAuditLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAuditLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuditLogListLogic {
	return &GetAuditLogListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAuditLogListLogic) GetAuditLogList(req *types.GetAuditLogListRequest) (*types.GetAuditLogListResponse, error) {
	pageNo, pageSize := req.PageNo, req.PageSize
	if pageNo == 0 {
		pageNo = l.svcCtx.Config.Page.PageNo
	}
	if pageSize == 0 {
		pageSize = l.svcCtx.Config.Page.PageSize
	}
	response, err := l.svcCtx.AppletAuditRPC.GetAuditLogList(l.ctx, &pb.GetAuditLogListRequest{
		PageRequest: &pb.PageRequest{PageNo: pageNo, PageSize: pageSize}, EventType: req.EventType,
		Module: req.Module, ActorID: req.ActorId, ActorName: req.ActorName, Result: req.Result,
		StartTime: req.StartTime, EndTime: req.EndTime,
	})
	if err != nil {
		return nil, err
	}
	list := make([]types.AuditLog, 0, len(response.List))
	for _, row := range response.List {
		list = append(list, types.AuditLog{
			ID: row.ID, ActorID: row.ActorID, ActorName: row.ActorName, AuthorityId: row.AuthorityId,
			Module: row.Module, Action: row.Action, Object: row.Object, Path: row.Path, Method: row.Method,
			Result: row.Result, StatusCode: row.StatusCode, IP: row.IP, TraceID: row.TraceID,
			DurationMs: row.DurationMs, Params: row.Params, CreatedAt: row.CreatedAt, EventType: row.EventType,
		})
	}
	return &types.GetAuditLogListResponse{List: list, PageResponse: types.PageResponse{PageNo: pageNo, PageSize: pageSize, Total: response.Total}}, nil
}
