package agenttoolslogic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"

	auditlogic "go-zero-admin/application/applet/rpc/internal/logic/audit"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	sessionlogic "go-zero-admin/application/applet/rpc/internal/logic/sessionmanage"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agenttools"
	"gorm.io/gorm"
)

func decodeArgs(raw json.RawMessage, target any) error {
	if len(raw) > 16*1024 || !utf8.Valid(raw) {
		return agenttools.ErrArguments
	}
	kind := reflect.TypeOf(target)
	if kind == nil || kind.Kind() != reflect.Pointer || kind.Elem().Kind() != reflect.Struct {
		return agenttools.ErrArguments
	}
	allowed := make(map[string]bool)
	for i := 0; i < kind.Elem().NumField(); i++ {
		name := strings.Split(kind.Elem().Field(i).Tag.Get("json"), ",")[0]
		if name != "" && name != "-" {
			allowed[name] = true
		}
	}
	// Disallow duplicate keys too: otherwise a model can hide invalid fields
	// behind last-value-wins semantics in a security sensitive argument object.
	dec := json.NewDecoder(bytes.NewReader(raw))
	first, err := dec.Token()
	if err != nil || first != json.Delim('{') {
		return agenttools.ErrArguments
	}
	seen := map[string]bool{}
	for dec.More() {
		token, err := dec.Token()
		if err != nil {
			return agenttools.ErrArguments
		}
		key, ok := token.(string)
		if !ok || seen[key] || !allowed[key] {
			return agenttools.ErrArguments
		}
		seen[key] = true
		var value json.RawMessage
		if dec.Decode(&value) != nil || bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			return agenttools.ErrArguments
		}
	}
	if _, err := dec.Token(); err != nil {
		return agenttools.ErrArguments
	}
	if _, err := dec.Token(); err != io.EOF {
		return agenttools.ErrArguments
	}
	typed := json.NewDecoder(bytes.NewReader(raw))
	typed.DisallowUnknownFields()
	if err := typed.Decode(target); err != nil {
		return agenttools.ErrArguments
	}
	return nil
}
func output(value any, summary string, count int) (agenttools.Output, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return agenttools.Output{}, err
	}
	return agenttools.Output{Content: string(raw), Summary: summary, Count: count}, nil
}

type auditArgs struct {
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Module    string `json:"module"`
	EventType string `json:"eventType"`
	Result    string `json:"result"`
}

func auditRange(args auditArgs, now time.Time) (time.Time, time.Time, error) {
	zone := time.FixedZone("Asia/Shanghai", 8*60*60)
	today := now.In(zone)
	start := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, zone)
	end := now
	var err error
	if args.StartTime != "" {
		start, err = time.Parse(time.RFC3339Nano, args.StartTime)
		if err != nil {
			return start, end, agenttools.ErrArguments
		}
	}
	if args.EndTime != "" {
		end, err = time.Parse(time.RFC3339Nano, args.EndTime)
		if err != nil {
			return start, end, agenttools.ErrArguments
		}
	}
	if start.After(end) || end.Sub(start) > 31*24*time.Hour || end.After(now.Add(time.Minute)) {
		return start, end, agenttools.ErrArguments
	}
	return start.UTC(), end.UTC(), nil
}
func queryAudit(ctx context.Context, s *svc.ServiceContext, raw json.RawMessage) (agenttools.Output, error) {
	var args auditArgs
	if err := decodeArgs(raw, &args); err != nil {
		return agenttools.Output{}, err
	}
	start, end, err := auditRange(args, time.Now())
	if err != nil {
		return agenttools.Output{}, err
	}
	req := &pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 20}, Module: args.Module, EventType: args.EventType, Result: args.Result, StartTime: timestamp(start), EndTime: timestamp(end)}
	r, err := auditlogic.NewGetAuditLogListLogic(ctx, s).GetAuditLogList(req)
	if err != nil {
		return agenttools.Output{}, err
	}
	// Aggregates cover the full authorized filter, not just the displayed sample.
	db := s.DB.WithContext(ctx).Model(&model.SysAuditLog{}).Where("created_at >= ? AND created_at <= ?", start, end)
	if args.Module != "" {
		db = db.Where("module = ?", strings.TrimSpace(args.Module))
	}
	if args.EventType != "" {
		db = db.Where("event_type = ?", args.EventType)
	}
	if args.Result != "" {
		db = db.Where("result = ?", args.Result)
	}
	counts := make([]struct {
		Module string `json:"module"`
		Count  int64  `json:"count"`
	}, 0)
	if err := db.Select("module, COUNT(*) AS count").Group("module").Order("count DESC,module ASC").Limit(100).Scan(&counts).Error; err != nil {
		return agenttools.Output{}, err
	}
	rows := make([]map[string]any, 0, len(r.List))
	for _, v := range r.List {
		rows = append(rows, map[string]any{"id": v.ID, "actorName": v.ActorName, "module": v.Module, "action": v.Action, "result": v.Result, "statusCode": v.StatusCode, "createdAt": v.CreatedAt})
	}
	return output(map[string]any{"startTime": timestamp(start), "endTime": timestamp(end), "total": r.Total, "byModule": counts, "moduleLimit": 100, "recent": rows, "recentLimit": 20}, fmt.Sprintf("匹配%d条审计日志，展示最近%d条", r.Total, len(rows)), len(rows))
}

func fileStatus(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest, raw json.RawMessage) (agenttools.Output, error) {
	var args struct {
		FileID int64 `json:"fileId"`
	}
	if err := decodeArgs(raw, &args); err != nil || args.FileID <= 0 {
		return agenttools.Output{}, agenttools.ErrArguments
	}
	q, err := datascope.Apply(ctx, s.DB.Model(&model.SysFileResource{}), a, "owner_id", "department_id")
	if err != nil {
		return agenttools.Output{}, err
	}
	var f model.SysFileResource
	if err := q.Where("id = ?", args.FileID).Take(&f).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return agenttools.Output{}, errors.New("file unavailable")
		}
		return agenttools.Output{}, err
	}
	var refs int64
	if err := s.DB.WithContext(ctx).Model(&model.SysFileReference{}).Where("file_id = ?", f.ID).Count(&refs).Error; err != nil {
		return agenttools.Output{}, err
	}
	// Role data ranges can change without replacing the login session. Reapply
	// the SQL scope before releasing a resource read under the earlier range.
	visible, err := datascope.Apply(ctx, s.DB.Model(&model.SysFileResource{}), a, "owner_id", "department_id")
	if err != nil {
		return agenttools.Output{}, err
	}
	var count int64
	if err := visible.Where("id = ?", f.ID).Count(&count).Error; err != nil {
		return agenttools.Output{}, err
	}
	if count != 1 {
		return agenttools.Output{}, errors.New("file unavailable")
	}
	reason := "状态正常，无引用；实际删除仍需使用文件模块并具备权限"
	if refs > 0 {
		reason = "文件存在业务引用，必须先在对应业务移除引用"
	} else if f.Status == "deleting" {
		reason = "删除尚未完成，可在文件模块重试"
	} else if f.Status == "deleted" {
		reason = "已删除"
	}
	return output(map[string]any{"id": f.ID, "name": f.Name, "mime": f.Mime, "size": f.Size, "status": f.Status, "visibility": f.Visibility, "references": refs, "explanation": reason}, "已查询文件状态及引用数", 1)
}
func myDevices(ctx context.Context, s *svc.ServiceContext, a *pb.SessionRequest, raw json.RawMessage) (agenttools.Output, error) {
	if err := decodeArgs(raw, &struct{}{}); err != nil {
		return agenttools.Output{}, err
	}
	r, err := sessionlogic.NewGetDeviceSessionsLogic(ctx, s).GetDeviceSessions(&pb.DeviceSessionListRequest{Actor: a, UserID: a.UserID, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 20}})
	if err != nil {
		return agenttools.Output{}, err
	}
	rows := make([]map[string]any, 0, len(r.List))
	for i, d := range r.List {
		rows = append(rows, map[string]any{"device": i + 1, "client": deviceClient(d.UserAgent), "current": d.Current, "createdAt": d.CreatedAt, "expiresAt": d.ExpiresAt})
	}
	return output(map[string]any{"total": r.Total, "items": rows, "limit": 20}, "已查询本人有效设备，隐藏会话编号与IP", len(rows))
}

func deviceClient(ua string) string {
	for _, browser := range []struct{ match, label string }{{"Edg/", "Edge"}, {"Firefox/", "Firefox"}, {"Chrome/", "Chrome"}, {"Safari/", "Safari"}} {
		if strings.Contains(ua, browser.match) {
			return browser.label
		}
	}
	return "其他客户端"
}
