package auditservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	where := "1=1"
	var args []interface{}
	if in.Module != "" {
		where += " AND module=?"
		args = append(args, in.Module)
	}
	if in.Action != "" {
		where += " AND action=?"
		args = append(args, in.Action)
	}
	if in.UserId > 0 {
		where += " AND user_id=?"
		args = append(args, in.UserId)
	}
	if in.Result > 0 {
		where += " AND result=?"
		args = append(args, in.Result)
	}
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM sys_audit_log WHERE "+where, args...)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	type auditRow struct {
		Id         int64  `db:"id"`
		UserId     int64  `db:"user_id"`
		Username   string `db:"username"`
		Module     string `db:"module"`
		Action     string `db:"action"`
		Method     string `db:"method"`
		Path       string `db:"path"`
		Params     string `db:"params"`
		Result     int32  `db:"result"`
		ErrorMsg   string `db:"error_msg"`
		IP         string `db:"ip"`
		UserAgent  string `db:"user_agent"`
		DurationMs int32  `db:"duration_ms"`
		CreatedAt  int64  `db:"created_at"`
	}
	var auditRows []auditRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &auditRows,
		"SELECT id, user_id, username, module, action, method, path, params, result, error_msg, ip, user_agent, duration_ms, created_at FROM sys_audit_log WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.AuditLogInfo, 0, len(auditRows))
	for _, r := range auditRows {
		list = append(list, &adminpb.AuditLogInfo{Id: r.Id, UserId: r.UserId, Username: r.Username, Module: r.Module, Action: r.Action, Method: r.Method, Path: r.Path, Params: r.Params, Result: r.Result, ErrorMsg: r.ErrorMsg, Ip: r.IP, UserAgent: r.UserAgent, DurationMs: r.DurationMs, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetAuditLogListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
