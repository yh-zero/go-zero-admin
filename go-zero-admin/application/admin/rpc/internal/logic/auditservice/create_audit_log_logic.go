package auditservicelogic

import (
	"context"
	"database/sql"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateAuditLogLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateAuditLogLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateAuditLogLogic {
	return &CreateAuditLogLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CreateAuditLogLogic) CreateAuditLog(in *adminpb.CreateAuditLogRequest) (*adminpb.NoDataResponse, error) {
	if in.Actor == nil || (in.Result != 1 && in.Result != 2) || in.Path == "" || in.DurationMs < 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err := l.svcCtx.SysAuditLogModel.Insert(l.ctx, &model.SysAuditLog{
		UserId:   sql.NullInt64{Int64: in.Actor.UserId, Valid: in.Actor.UserId > 0},
		Username: auditText(in.Actor.Username, 64), Module: auditText(in.Module, 64), Action: auditText(in.Action, 32),
		Method: auditText(in.Method, 16), Path: auditText(in.Path, 255), Result: int64(in.Result), ErrorMsg: auditText(in.ErrorMsg, 512),
		Ip: auditText(in.Actor.Ip, 64), UserAgent: auditText(in.Actor.UserAgent, 512), DurationMs: uint64(in.DurationMs),
	})
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}

func auditText(value string, limit int) string {
	chars := []rune(value)
	if len(chars) > limit {
		return string(chars[:limit])
	}
	return value
}
