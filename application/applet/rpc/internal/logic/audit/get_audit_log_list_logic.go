package auditlogic

import (
	"context"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
)

type GetAuditLogListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetAuditLogListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuditLogListLogic {
	return &GetAuditLogListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetAuditLogListLogic) GetAuditLogList(in *pb.GetAuditLogListRequest) (*pb.GetAuditLogListResponse, error) {
	if in == nil || in.PageRequest == nil {
		return nil, invalidFilter()
	}
	offset, limit, err := accessutil.Page(in.PageRequest.PageNo, in.PageRequest.PageSize)
	if err != nil {
		return nil, err
	}
	if in.ActorID < 0 || (in.EventType != "" && in.EventType != "operation" && in.EventType != "login") ||
		(in.Result != "" && in.Result != "success" && in.Result != "failure") ||
		len([]rune(in.Module)) > 64 || len([]rune(in.ActorName)) > 64 {
		return nil, invalidFilter()
	}
	start, err := parseTime(in.StartTime)
	if err != nil {
		return nil, invalidFilter()
	}
	end, err := parseTime(in.EndTime)
	if err != nil || (!start.IsZero() && !end.IsZero() && start.After(end)) {
		return nil, invalidFilter()
	}
	db := l.svcCtx.DB.WithContext(l.ctx).Model(&model.SysAuditLog{})
	if in.EventType != "" {
		db = db.Where("event_type = ?", in.EventType)
	}
	if in.Module != "" {
		db = db.Where("module = ?", strings.TrimSpace(in.Module))
	}
	if in.ActorID > 0 {
		db = db.Where("actor_id = ?", in.ActorID)
	}
	if in.ActorName != "" {
		db = db.Where("actor_name = ?", strings.TrimSpace(in.ActorName))
	}
	if in.Result != "" {
		db = db.Where("result = ?", in.Result)
	}
	if !start.IsZero() {
		db = db.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		db = db.Where("created_at <= ?", end)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []model.SysAuditLog
	if err := db.Order("created_at DESC, id DESC").Offset(offset).Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	response := &pb.GetAuditLogListResponse{Total: total, List: make([]*pb.AuditLog, 0, len(rows))}
	for _, row := range rows {
		response.List = append(response.List, &pb.AuditLog{
			ID: row.ID, ActorID: row.ActorID, ActorName: row.ActorName, AuthorityId: row.AuthorityID,
			Module: row.Module, Action: row.Action, Object: row.Object, Path: row.Path, Method: row.Method,
			Result: row.Result, StatusCode: row.StatusCode, IP: row.IP, TraceID: row.TraceID,
			DurationMs: row.DurationMs, Params: row.Params, CreatedAt: row.CreatedAt.UTC().Format(time.RFC3339Nano), EventType: row.EventType,
		})
	}
	return response, nil
}

func invalidFilter() error {
	return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "审计查询参数无效，时间须为RFC3339格式且开始时间不能晚于结束时间")
}
func parseTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	return parsed.UTC(), err
}
