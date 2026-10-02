package auditlogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/result/xerr"
)

type RecordAuditLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRecordAuditLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RecordAuditLogic {
	return &RecordAuditLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *RecordAuditLogic) RecordAudit(in *pb.RecordAuditRequest) (*pb.NoDataResponse, error) {
	if in == nil || in.Log == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "审计事件不能为空")
	}
	log := in.Log
	event := audit.Event{
		ActorID: log.ActorID, ActorName: log.ActorName, AuthorityID: log.AuthorityId,
		Module: log.Module, Action: log.Action, Object: log.Object, Path: log.Path,
		Method: log.Method, Result: log.Result, StatusCode: log.StatusCode,
		IP: log.IP, TraceID: log.TraceID, DurationMs: log.DurationMs,
		Params: log.Params, EventType: log.EventType,
	}
	if err := audit.Normalize(&event); err != nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "审计事件参数无效")
	}
	if err := audit.Record(l.ctx, l.svcCtx.DB.DB, event); err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}
