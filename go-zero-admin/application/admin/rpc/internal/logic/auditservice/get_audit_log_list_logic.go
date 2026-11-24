package auditservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetAuditLogListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuditLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuditLogListLogic {
	return &GetAuditLogListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuditLogListLogic) GetAuditLogList(in *adminpb.GetAuditLogListRequest) (*adminpb.GetAuditLogListResponse, error) {
	rows, total, err := l.svcCtx.SysAuditLogModel.ListAuditLogPage(l.ctx, model.AuditLogListFilter{
		Module: in.Module, Action: in.Action, UserId: in.UserId, Result: int64(in.Result),
		Username: in.Username, Method: in.Method, Path: in.Path, IP: in.Ip,
		StartTime: in.StartTime, EndTime: in.EndTime,
		MinDurationMs: in.MinDurationMs, MaxDurationMs: in.MaxDurationMs,
	}, in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if errors.Is(err, model.ErrInvalidAuditFilter) {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "审计筛选时间或耗时范围不合法")
	}
	if err != nil {
		l.Errorf("query audit logs error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.AuditLogInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.AuditLogInfo{
			Id: r.Id, UserId: r.UserId, Username: r.Username, Module: r.Module, Action: r.Action,
			Method: r.Method, Path: r.Path, Params: r.Params, Result: r.Result, ErrorMsg: r.ErrorMsg,
			Ip: r.IP, UserAgent: r.UserAgent, DurationMs: r.DurationMs, CreatedAt: r.CreatedAt,
		})
	}
	return &adminpb.GetAuditLogListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
